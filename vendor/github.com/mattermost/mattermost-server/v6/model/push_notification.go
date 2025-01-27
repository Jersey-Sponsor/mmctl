// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package model

import (
	"encoding/json"
	"errors"
	"io"
	"strings"
)

const (
	PushNotifyApple              = "apple"
	PushNotifyAndroid            = "android"
	PushNotifyAppleReactNative   = "apple_rn"
	PushNotifyAndroidReactNative = "android_rn"

	PushTypeMessage     = "message"
	PushTypeClear       = "clear"
	PushTypeUpdateBadge = "update_badge"
	PushTypeSession     = "session"
	PushMessageV2       = "v2"

	PushSoundNone = "none"

	// The category is set to handle a set of interactive Actions
	// with the push notifications
	CategoryCanReply = "CAN_REPLY"

	MHPNS = "https://push.mattermost.com"

	PushSendPrepare = "Prepared to send"
	PushSendSuccess = "Successful"
	PushNotSent     = "Not Sent due to preferences"
	PushReceived    = "Received by device"
)

type PushNotificationAck struct {
	Id               string `json:"id"`
	ClientReceivedAt int64  `json:"received_at"`
	ClientPlatform   string `json:"platform"`
	NotificationType string `json:"type"`
	PostId           string `json:"post_id,omitempty"`
	IsIdLoaded       bool   `json:"is_id_loaded"`
}

type PushNotification struct {
	AckId            string `json:"ack_id"`
	Platform         string `json:"platform"`
	ServerId         string `json:"server_id"`
	DeviceId         string `json:"device_id"`
	PostId           string `json:"post_id"`
	Category         string `json:"category,omitempty"`
	Sound            string `json:"sound,omitempty"`
	Message          string `json:"message,omitempty"`
	Badge            int    `json:"badge,omitempty"`
	ContentAvailable int    `json:"cont_ava,omitempty"`
	TeamId           string `json:"team_id,omitempty"`
	ChannelId        string `json:"channel_id,omitempty"`
	RootId           string `json:"root_id,omitempty"`
	ChannelName      string `json:"channel_name,omitempty"`
	Type             string `json:"type,omitempty"`
	SenderId         string `json:"sender_id,omitempty"`
	SenderName       string `json:"sender_name,omitempty"`
	OverrideUsername string `json:"override_username,omitempty"`
	OverrideIconURL  string `json:"override_icon_url,omitempty"`
	FromWebhook      string `json:"from_webhook,omitempty"`
	Version          string `json:"version,omitempty"`
	IsIdLoaded       bool   `json:"is_id_loaded"`
}

func (pn *PushNotification) ToJson() string {
	b, _ := json.Marshal(pn)
	return string(b)
}

func (pn *PushNotification) DeepCopy() *PushNotification {
	copy := *pn
	return &copy
}

func (pn *PushNotification) SetDeviceIdAndPlatform(deviceId string) {

	index := strings.Index(deviceId, ":")

	if index > -1 {
		pn.Platform = deviceId[:index]
		pn.DeviceId = deviceId[index+1:]
	}
}

func PushNotificationFromJson(data io.Reader) (*PushNotification, error) {
	if data == nil {
		return nil, errors.New("push notification data can't be nil")
	}
	var pn *PushNotification
	if err := json.NewDecoder(data).Decode(&pn); err != nil {
		return nil, err
	}
	return pn, nil
}

func PushNotificationAckFromJson(data io.Reader) (*PushNotificationAck, error) {
	if data == nil {
		return nil, errors.New("push notification data can't be nil")
	}
	var ack *PushNotificationAck
	if err := json.NewDecoder(data).Decode(&ack); err != nil {
		return nil, err
	}
	return ack, nil
}

func (ack *PushNotificationAck) ToJson() string {
	b, _ := json.Marshal(ack)
	return string(b)
}
