// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package sqlstore

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/store"
)

const (
	sessionsCleanupDelay = 100 * time.Millisecond
)

type SqlSessionStore struct {
	*SqlStore
}

func newSqlSessionStore(sqlStore *SqlStore) store.SessionStore {
	us := &SqlSessionStore{sqlStore}

	for _, db := range sqlStore.GetAllConns() {
		table := db.AddTableWithName(model.Session{}, "Sessions").SetKeys(false, "Id")
		table.ColMap("Id").SetMaxSize(26)
		table.ColMap("Token").SetMaxSize(26)
		table.ColMap("UserId").SetMaxSize(26)
		table.ColMap("DeviceId").SetMaxSize(512)
		table.ColMap("Roles").SetMaxSize(64)
		table.ColMap("Props").SetDataType(sqlStore.jsonDataType())
	}

	return us
}

func (me SqlSessionStore) createIndexesIfNotExists() {
	me.CreateIndexIfNotExists("idx_sessions_user_id", "Sessions", "UserId")
	me.CreateIndexIfNotExists("idx_sessions_token", "Sessions", "Token")
	me.CreateIndexIfNotExists("idx_sessions_expires_at", "Sessions", "ExpiresAt")
	me.CreateIndexIfNotExists("idx_sessions_create_at", "Sessions", "CreateAt")
	me.CreateIndexIfNotExists("idx_sessions_last_activity_at", "Sessions", "LastActivityAt")
}

func (me SqlSessionStore) Save(session *model.Session) (*model.Session, error) {
	if session.Id != "" {
		return nil, store.NewErrInvalidInput("Session", "id", session.Id)
	}
	session.PreSave()

	if err := me.GetMaster().Insert(session); err != nil {
		return nil, errors.Wrapf(err, "failed to save Session with id=%s", session.Id)
	}

	teamMembers, err := me.Team().GetTeamsForUser(context.Background(), session.UserId)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find TeamMembers for Session with userId=%s", session.UserId)
	}

	session.TeamMembers = make([]*model.TeamMember, 0, len(teamMembers))
	for _, tm := range teamMembers {
		if tm.DeleteAt == 0 {
			session.TeamMembers = append(session.TeamMembers, tm)
		}
	}

	return session, nil
}

func (me SqlSessionStore) Get(ctx context.Context, sessionIdOrToken string) (*model.Session, error) {
	var sessions []*model.Session

	if _, err := me.DBFromContext(ctx).Select(&sessions, "SELECT * FROM Sessions WHERE Token = :Token OR Id = :Id LIMIT 1", map[string]interface{}{"Token": sessionIdOrToken, "Id": sessionIdOrToken}); err != nil {
		return nil, errors.Wrapf(err, "failed to find Sessions with sessionIdOrToken=%s", sessionIdOrToken)
	} else if len(sessions) == 0 {
		return nil, store.NewErrNotFound("Session", fmt.Sprintf("sessionIdOrToken=%s", sessionIdOrToken))
	}
	session := sessions[0]

	tempMembers, err := me.Team().GetTeamsForUser(
		WithMaster(context.Background()),
		session.UserId)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find TeamMembers for Session with userId=%s", session.UserId)
	}
	sessions[0].TeamMembers = make([]*model.TeamMember, 0, len(tempMembers))
	for _, tm := range tempMembers {
		if tm.DeleteAt == 0 {
			sessions[0].TeamMembers = append(sessions[0].TeamMembers, tm)
		}
	}
	return session, nil
}

func (me SqlSessionStore) GetSessions(userId string) ([]*model.Session, error) {
	var sessions []*model.Session

	if _, err := me.GetReplica().Select(&sessions, "SELECT * FROM Sessions WHERE UserId = :UserId ORDER BY LastActivityAt DESC", map[string]interface{}{"UserId": userId}); err != nil {
		return nil, errors.Wrapf(err, "failed to find Sessions with userId=%s", userId)
	}

	teamMembers, err := me.Team().GetTeamsForUser(context.Background(), userId)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find TeamMembers for Session with userId=%s", userId)
	}

	for _, session := range sessions {
		session.TeamMembers = make([]*model.TeamMember, 0, len(teamMembers))
		for _, tm := range teamMembers {
			if tm.DeleteAt == 0 {
				session.TeamMembers = append(session.TeamMembers, tm)
			}
		}
	}
	return sessions, nil
}

func (me SqlSessionStore) GetSessionsWithActiveDeviceIds(userId string) ([]*model.Session, error) {
	query :=
		`SELECT *
		FROM
			Sessions
		WHERE
			UserId = :UserId AND
			ExpiresAt != 0 AND
			:ExpiresAt <= ExpiresAt AND
			DeviceId != ''`

	var sessions []*model.Session

	_, err := me.GetReplica().Select(&sessions, query, map[string]interface{}{"UserId": userId, "ExpiresAt": model.GetMillis()})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find Sessions with userId=%s", userId)
	}
	return sessions, nil
}

func (me SqlSessionStore) GetSessionsExpired(thresholdMillis int64, mobileOnly bool, unnotifiedOnly bool) ([]*model.Session, error) {
	now := model.GetMillis()
	builder := me.getQueryBuilder().
		Select("*").
		From("Sessions").
		Where(sq.NotEq{"ExpiresAt": 0}).
		Where(sq.Lt{"ExpiresAt": now}).
		Where(sq.Gt{"ExpiresAt": now - thresholdMillis})
	if mobileOnly {
		builder = builder.Where(sq.NotEq{"DeviceId": ""})
	}
	if unnotifiedOnly {
		builder = builder.Where(sq.NotEq{"ExpiredNotify": true})
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "sessions_tosql")
	}

	var sessions []*model.Session

	_, err = me.GetReplica().Select(&sessions, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find Sessions")
	}
	return sessions, nil
}

func (me SqlSessionStore) UpdateExpiredNotify(sessionId string, notified bool) error {
	query, args, err := me.getQueryBuilder().
		Update("Sessions").
		Set("ExpiredNotify", notified).
		Where(sq.Eq{"Id": sessionId}).
		ToSql()
	if err != nil {
		return errors.Wrap(err, "sessions_tosql")
	}

	_, err = me.GetMaster().Exec(query, args...)
	if err != nil {
		return errors.Wrapf(err, "failed to update Session with id=%s", sessionId)
	}
	return nil
}

func (me SqlSessionStore) Remove(sessionIdOrToken string) error {
	_, err := me.GetMaster().Exec("DELETE FROM Sessions WHERE Id = :Id Or Token = :Token", map[string]interface{}{"Id": sessionIdOrToken, "Token": sessionIdOrToken})
	if err != nil {
		return errors.Wrapf(err, "failed to delete Session with sessionIdOrToken=%s", sessionIdOrToken)
	}
	return nil
}

func (me SqlSessionStore) RemoveAllSessions() error {
	_, err := me.GetMaster().Exec("DELETE FROM Sessions")
	if err != nil {
		return errors.Wrap(err, "failed to delete all Sessions")
	}
	return nil
}

func (me SqlSessionStore) PermanentDeleteSessionsByUser(userId string) error {
	_, err := me.GetMaster().Exec("DELETE FROM Sessions WHERE UserId = :UserId", map[string]interface{}{"UserId": userId})
	if err != nil {
		return errors.Wrapf(err, "failed to delete Session with userId=%s", userId)
	}

	return nil
}

func (me SqlSessionStore) UpdateExpiresAt(sessionId string, time int64) error {
	_, err := me.GetMaster().Exec("UPDATE Sessions SET ExpiresAt = :ExpiresAt, ExpiredNotify = false WHERE Id = :Id", map[string]interface{}{"ExpiresAt": time, "Id": sessionId})
	if err != nil {
		return errors.Wrapf(err, "failed to update Session with sessionId=%s", sessionId)
	}
	return nil
}

func (me SqlSessionStore) UpdateLastActivityAt(sessionId string, time int64) error {
	_, err := me.GetMaster().Exec("UPDATE Sessions SET LastActivityAt = :LastActivityAt WHERE Id = :Id", map[string]interface{}{"LastActivityAt": time, "Id": sessionId})
	if err != nil {
		return errors.Wrapf(err, "failed to update Session with id=%s", sessionId)
	}
	return nil
}

func (me SqlSessionStore) UpdateRoles(userId, roles string) (string, error) {
	query := "UPDATE Sessions SET Roles = :Roles WHERE UserId = :UserId"

	_, err := me.GetMaster().Exec(query, map[string]interface{}{"Roles": roles, "UserId": userId})
	if err != nil {
		return "", errors.Wrapf(err, "failed to update Session with userId=%s and roles=%s", userId, roles)
	}
	return userId, nil
}

func (me SqlSessionStore) UpdateDeviceId(id string, deviceId string, expiresAt int64) (string, error) {
	query := "UPDATE Sessions SET DeviceId = :DeviceId, ExpiresAt = :ExpiresAt, ExpiredNotify = false WHERE Id = :Id"

	_, err := me.GetMaster().Exec(query, map[string]interface{}{"DeviceId": deviceId, "Id": id, "ExpiresAt": expiresAt})
	if err != nil {
		return "", errors.Wrapf(err, "failed to update Session with id=%s", id)
	}
	return deviceId, nil
}

func (me SqlSessionStore) UpdateProps(session *model.Session) error {
	_, err := me.GetMaster().Exec(`UPDATE Sessions
		SET Props=:Props
		WHERE Id=:Id`, map[string]interface{}{
		"Props": model.MapToJson(session.Props),
		"Id":    session.Id,
	})
	if err != nil {
		return errors.Wrap(err, "failed to update Session")
	}
	return nil
}

func (me SqlSessionStore) AnalyticsSessionCount() (int64, error) {
	query :=
		`SELECT
			COUNT(*)
		FROM
			Sessions
		WHERE ExpiresAt > :Time`
	count, err := me.GetReplica().SelectInt(query, map[string]interface{}{"Time": model.GetMillis()})
	if err != nil {
		return int64(0), errors.Wrap(err, "failed to count Sessions")
	}
	return count, nil
}

func (me SqlSessionStore) Cleanup(expiryTime int64, batchSize int64) error {
	var query string
	if me.DriverName() == model.DatabaseDriverPostgres {
		query = "DELETE FROM Sessions WHERE Id IN (SELECT Id FROM Sessions WHERE ExpiresAt != 0 AND :ExpiresAt > ExpiresAt LIMIT :Limit)"
	} else {
		query = "DELETE FROM Sessions WHERE ExpiresAt != 0 AND :ExpiresAt > ExpiresAt LIMIT :Limit"
	}

	var rowsAffected int64 = 1

	for rowsAffected > 0 {
		sqlResult, err := me.GetMaster().Exec(query, map[string]interface{}{"ExpiresAt": expiryTime, "Limit": batchSize})
		if err != nil {
			return errors.Wrap(err, "unable to delete sessions")
		}
		var rowErr error
		rowsAffected, rowErr = sqlResult.RowsAffected()
		if rowErr != nil {
			return errors.Wrap(err, "unable to delete sessions")
		}

		time.Sleep(sessionsCleanupDelay)
	}

	return nil
}
