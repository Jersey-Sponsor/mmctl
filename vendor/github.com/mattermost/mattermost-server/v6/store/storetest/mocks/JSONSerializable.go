// Code generated by mockery v1.0.0. DO NOT EDIT.

// Regenerate this file using `make store-mocks`.

package mocks

import mock "github.com/stretchr/testify/mock"

// JSONSerializable is an autogenerated mock type for the JSONSerializable type
type JSONSerializable struct {
	mock.Mock
}

// ToJSON provides a mock function with given fields:
func (_m *JSONSerializable) ToJSON() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}
