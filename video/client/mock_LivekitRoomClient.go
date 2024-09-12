// Code generated by mockery v2.45.1. DO NOT EDIT.

package client

import (
	context "context"

	livekit "github.com/livekit/protocol/livekit"
	mock "github.com/stretchr/testify/mock"
)

// MockLivekitRoomClient is an autogenerated mock type for the LivekitRoomClient type
type MockLivekitRoomClient struct {
	mock.Mock
}

type MockLivekitRoomClient_Expecter struct {
	mock *mock.Mock
}

func (_m *MockLivekitRoomClient) EXPECT() *MockLivekitRoomClient_Expecter {
	return &MockLivekitRoomClient_Expecter{mock: &_m.Mock}
}

// ListParticipants provides a mock function with given fields: ctx, req
func (_m *MockLivekitRoomClient) ListParticipants(ctx context.Context, req *livekit.ListParticipantsRequest) (*livekit.ListParticipantsResponse, error) {
	ret := _m.Called(ctx, req)

	if len(ret) == 0 {
		panic("no return value specified for ListParticipants")
	}

	var r0 *livekit.ListParticipantsResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *livekit.ListParticipantsRequest) (*livekit.ListParticipantsResponse, error)); ok {
		return rf(ctx, req)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *livekit.ListParticipantsRequest) *livekit.ListParticipantsResponse); ok {
		r0 = rf(ctx, req)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*livekit.ListParticipantsResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *livekit.ListParticipantsRequest) error); ok {
		r1 = rf(ctx, req)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockLivekitRoomClient_ListParticipants_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ListParticipants'
type MockLivekitRoomClient_ListParticipants_Call struct {
	*mock.Call
}

// ListParticipants is a helper method to define mock.On call
//   - ctx context.Context
//   - req *livekit.ListParticipantsRequest
func (_e *MockLivekitRoomClient_Expecter) ListParticipants(ctx interface{}, req interface{}) *MockLivekitRoomClient_ListParticipants_Call {
	return &MockLivekitRoomClient_ListParticipants_Call{Call: _e.mock.On("ListParticipants", ctx, req)}
}

func (_c *MockLivekitRoomClient_ListParticipants_Call) Run(run func(ctx context.Context, req *livekit.ListParticipantsRequest)) *MockLivekitRoomClient_ListParticipants_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*livekit.ListParticipantsRequest))
	})
	return _c
}

func (_c *MockLivekitRoomClient_ListParticipants_Call) Return(_a0 *livekit.ListParticipantsResponse, _a1 error) *MockLivekitRoomClient_ListParticipants_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockLivekitRoomClient_ListParticipants_Call) RunAndReturn(run func(context.Context, *livekit.ListParticipantsRequest) (*livekit.ListParticipantsResponse, error)) *MockLivekitRoomClient_ListParticipants_Call {
	_c.Call.Return(run)
	return _c
}

// ListRooms provides a mock function with given fields: ctx, req
func (_m *MockLivekitRoomClient) ListRooms(ctx context.Context, req *livekit.ListRoomsRequest) (*livekit.ListRoomsResponse, error) {
	ret := _m.Called(ctx, req)

	if len(ret) == 0 {
		panic("no return value specified for ListRooms")
	}

	var r0 *livekit.ListRoomsResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *livekit.ListRoomsRequest) (*livekit.ListRoomsResponse, error)); ok {
		return rf(ctx, req)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *livekit.ListRoomsRequest) *livekit.ListRoomsResponse); ok {
		r0 = rf(ctx, req)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*livekit.ListRoomsResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *livekit.ListRoomsRequest) error); ok {
		r1 = rf(ctx, req)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockLivekitRoomClient_ListRooms_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ListRooms'
type MockLivekitRoomClient_ListRooms_Call struct {
	*mock.Call
}

// ListRooms is a helper method to define mock.On call
//   - ctx context.Context
//   - req *livekit.ListRoomsRequest
func (_e *MockLivekitRoomClient_Expecter) ListRooms(ctx interface{}, req interface{}) *MockLivekitRoomClient_ListRooms_Call {
	return &MockLivekitRoomClient_ListRooms_Call{Call: _e.mock.On("ListRooms", ctx, req)}
}

func (_c *MockLivekitRoomClient_ListRooms_Call) Run(run func(ctx context.Context, req *livekit.ListRoomsRequest)) *MockLivekitRoomClient_ListRooms_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*livekit.ListRoomsRequest))
	})
	return _c
}

func (_c *MockLivekitRoomClient_ListRooms_Call) Return(_a0 *livekit.ListRoomsResponse, _a1 error) *MockLivekitRoomClient_ListRooms_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockLivekitRoomClient_ListRooms_Call) RunAndReturn(run func(context.Context, *livekit.ListRoomsRequest) (*livekit.ListRoomsResponse, error)) *MockLivekitRoomClient_ListRooms_Call {
	_c.Call.Return(run)
	return _c
}

// MutePublishedTrack provides a mock function with given fields: ctx, req
func (_m *MockLivekitRoomClient) MutePublishedTrack(ctx context.Context, req *livekit.MuteRoomTrackRequest) (*livekit.MuteRoomTrackResponse, error) {
	ret := _m.Called(ctx, req)

	if len(ret) == 0 {
		panic("no return value specified for MutePublishedTrack")
	}

	var r0 *livekit.MuteRoomTrackResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *livekit.MuteRoomTrackRequest) (*livekit.MuteRoomTrackResponse, error)); ok {
		return rf(ctx, req)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *livekit.MuteRoomTrackRequest) *livekit.MuteRoomTrackResponse); ok {
		r0 = rf(ctx, req)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*livekit.MuteRoomTrackResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *livekit.MuteRoomTrackRequest) error); ok {
		r1 = rf(ctx, req)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockLivekitRoomClient_MutePublishedTrack_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'MutePublishedTrack'
type MockLivekitRoomClient_MutePublishedTrack_Call struct {
	*mock.Call
}

// MutePublishedTrack is a helper method to define mock.On call
//   - ctx context.Context
//   - req *livekit.MuteRoomTrackRequest
func (_e *MockLivekitRoomClient_Expecter) MutePublishedTrack(ctx interface{}, req interface{}) *MockLivekitRoomClient_MutePublishedTrack_Call {
	return &MockLivekitRoomClient_MutePublishedTrack_Call{Call: _e.mock.On("MutePublishedTrack", ctx, req)}
}

func (_c *MockLivekitRoomClient_MutePublishedTrack_Call) Run(run func(ctx context.Context, req *livekit.MuteRoomTrackRequest)) *MockLivekitRoomClient_MutePublishedTrack_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*livekit.MuteRoomTrackRequest))
	})
	return _c
}

func (_c *MockLivekitRoomClient_MutePublishedTrack_Call) Return(_a0 *livekit.MuteRoomTrackResponse, _a1 error) *MockLivekitRoomClient_MutePublishedTrack_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockLivekitRoomClient_MutePublishedTrack_Call) RunAndReturn(run func(context.Context, *livekit.MuteRoomTrackRequest) (*livekit.MuteRoomTrackResponse, error)) *MockLivekitRoomClient_MutePublishedTrack_Call {
	_c.Call.Return(run)
	return _c
}

// RemoveParticipant provides a mock function with given fields: ctx, req
func (_m *MockLivekitRoomClient) RemoveParticipant(ctx context.Context, req *livekit.RoomParticipantIdentity) (*livekit.RemoveParticipantResponse, error) {
	ret := _m.Called(ctx, req)

	if len(ret) == 0 {
		panic("no return value specified for RemoveParticipant")
	}

	var r0 *livekit.RemoveParticipantResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *livekit.RoomParticipantIdentity) (*livekit.RemoveParticipantResponse, error)); ok {
		return rf(ctx, req)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *livekit.RoomParticipantIdentity) *livekit.RemoveParticipantResponse); ok {
		r0 = rf(ctx, req)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*livekit.RemoveParticipantResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *livekit.RoomParticipantIdentity) error); ok {
		r1 = rf(ctx, req)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockLivekitRoomClient_RemoveParticipant_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'RemoveParticipant'
type MockLivekitRoomClient_RemoveParticipant_Call struct {
	*mock.Call
}

// RemoveParticipant is a helper method to define mock.On call
//   - ctx context.Context
//   - req *livekit.RoomParticipantIdentity
func (_e *MockLivekitRoomClient_Expecter) RemoveParticipant(ctx interface{}, req interface{}) *MockLivekitRoomClient_RemoveParticipant_Call {
	return &MockLivekitRoomClient_RemoveParticipant_Call{Call: _e.mock.On("RemoveParticipant", ctx, req)}
}

func (_c *MockLivekitRoomClient_RemoveParticipant_Call) Run(run func(ctx context.Context, req *livekit.RoomParticipantIdentity)) *MockLivekitRoomClient_RemoveParticipant_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*livekit.RoomParticipantIdentity))
	})
	return _c
}

func (_c *MockLivekitRoomClient_RemoveParticipant_Call) Return(_a0 *livekit.RemoveParticipantResponse, _a1 error) *MockLivekitRoomClient_RemoveParticipant_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockLivekitRoomClient_RemoveParticipant_Call) RunAndReturn(run func(context.Context, *livekit.RoomParticipantIdentity) (*livekit.RemoveParticipantResponse, error)) *MockLivekitRoomClient_RemoveParticipant_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockLivekitRoomClient creates a new instance of MockLivekitRoomClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockLivekitRoomClient(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockLivekitRoomClient {
	mock := &MockLivekitRoomClient{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
