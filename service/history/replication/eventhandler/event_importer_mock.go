// The MIT License
//
// Copyright (c) 2020 Temporal Technologies Inc.  All rights reserved.
//
// Copyright (c) 2020 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

// Code generated by MockGen. DO NOT EDIT.
// Source: event_importer.go

// Package eventhandler is a generated GoMock package.
package eventhandler

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	definition "go.temporal.io/server/common/definition"
)

// MockEventImporter is a mock of EventImporter interface.
type MockEventImporter struct {
	ctrl     *gomock.Controller
	recorder *MockEventImporterMockRecorder
}

// MockEventImporterMockRecorder is the mock recorder for MockEventImporter.
type MockEventImporterMockRecorder struct {
	mock *MockEventImporter
}

// NewMockEventImporter creates a new mock instance.
func NewMockEventImporter(ctrl *gomock.Controller) *MockEventImporter {
	mock := &MockEventImporter{ctrl: ctrl}
	mock.recorder = &MockEventImporterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockEventImporter) EXPECT() *MockEventImporterMockRecorder {
	return m.recorder
}

// ImportHistoryEventsFromBeginning mocks base method.
func (m *MockEventImporter) ImportHistoryEventsFromBeginning(ctx context.Context, remoteCluster string, workflowKey definition.WorkflowKey, endEventId, endEventVersion int64) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ImportHistoryEventsFromBeginning", ctx, remoteCluster, workflowKey, endEventId, endEventVersion)
	ret0, _ := ret[0].(error)
	return ret0
}

// ImportHistoryEventsFromBeginning indicates an expected call of ImportHistoryEventsFromBeginning.
func (mr *MockEventImporterMockRecorder) ImportHistoryEventsFromBeginning(ctx, remoteCluster, workflowKey, endEventId, endEventVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ImportHistoryEventsFromBeginning", reflect.TypeOf((*MockEventImporter)(nil).ImportHistoryEventsFromBeginning), ctx, remoteCluster, workflowKey, endEventId, endEventVersion)
}
