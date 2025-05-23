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

package shard

import (
	"context"
	"time"

	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/server/api/adminservice/v1"
	clockspb "go.temporal.io/server/api/clock/v1"
	"go.temporal.io/server/api/historyservice/v1"
	persistencespb "go.temporal.io/server/api/persistence/v1"
	"go.temporal.io/server/common/archiver"
	"go.temporal.io/server/common/clock"
	"go.temporal.io/server/common/cluster"
	"go.temporal.io/server/common/definition"
	"go.temporal.io/server/common/log"
	"go.temporal.io/server/common/metrics"
	"go.temporal.io/server/common/namespace"
	"go.temporal.io/server/common/persistence"
	"go.temporal.io/server/common/persistence/serialization"
	"go.temporal.io/server/common/pingable"
	"go.temporal.io/server/common/searchattribute"
	"go.temporal.io/server/service/history/configs"
	"go.temporal.io/server/service/history/events"
	"go.temporal.io/server/service/history/hsm"
	"go.temporal.io/server/service/history/tasks"
)

//go:generate mockgen -copyright_file ../../../LICENSE -package $GOPACKAGE -source $GOFILE -destination context_mock.go

type (
	// Context represents a history engine shard
	Context interface {
		GetShardID() int32
		GetRangeID() int64
		GetOwner() string
		GetExecutionManager() persistence.ExecutionManager
		GetNamespaceRegistry() namespace.Registry
		GetClusterMetadata() cluster.Metadata
		GetConfig() *configs.Config
		GetEventsCache() events.Cache
		GetLogger() log.Logger
		GetThrottledLogger() log.Logger
		GetMetricsHandler() metrics.Handler
		GetTimeSource() clock.TimeSource

		GetRemoteAdminClient(string) (adminservice.AdminServiceClient, error)
		GetHistoryClient() historyservice.HistoryServiceClient
		GetPayloadSerializer() serialization.Serializer

		GetSearchAttributesProvider() searchattribute.Provider
		GetSearchAttributesMapperProvider() searchattribute.MapperProvider
		GetArchivalMetadata() archiver.ArchivalMetadata

		GetEngine(ctx context.Context) (Engine, error)

		AssertOwnership(ctx context.Context) error
		NewVectorClock() (*clockspb.VectorClock, error)
		CurrentVectorClock() *clockspb.VectorClock

		GenerateTaskID() (int64, error)
		GenerateTaskIDs(number int) ([]int64, error)

		GetQueueExclusiveHighReadWatermark(category tasks.Category) tasks.Key
		GetQueueState(category tasks.Category) (*persistencespb.QueueState, bool)
		SetQueueState(category tasks.Category, tasksCompleted int, state *persistencespb.QueueState) error
		UpdateReplicationQueueReaderState(readerID int64, readerState *persistencespb.QueueReaderState) error

		GetReplicatorDLQAckLevel(sourceCluster string) int64
		UpdateReplicatorDLQAckLevel(sourCluster string, ackLevel int64) error

		UpdateRemoteClusterInfo(cluster string, ackTaskID int64, ackTimestamp time.Time)
		UpdateRemoteReaderInfo(readerID int64, ackTaskID int64, ackTimestamp time.Time) error

		SetCurrentTime(cluster string, currentTime time.Time)
		GetCurrentTime(cluster string) time.Time

		GetReplicationStatus(cluster []string) (map[string]*historyservice.ShardReplicationStatusPerCluster, map[string]*historyservice.HandoverNamespaceInfo, error)

		UpdateHandoverNamespace(ns *namespace.Namespace, deletedFromDb bool)

		AppendHistoryEvents(ctx context.Context, request *persistence.AppendHistoryNodesRequest, namespaceID namespace.ID, execution *commonpb.WorkflowExecution) (int, error)

		AddTasks(ctx context.Context, request *persistence.AddHistoryTasksRequest) error
		AddSpeculativeWorkflowTaskTimeoutTask(task *tasks.WorkflowTaskTimeoutTask) error
		CreateWorkflowExecution(ctx context.Context, request *persistence.CreateWorkflowExecutionRequest) (*persistence.CreateWorkflowExecutionResponse, error)
		UpdateWorkflowExecution(ctx context.Context, request *persistence.UpdateWorkflowExecutionRequest) (*persistence.UpdateWorkflowExecutionResponse, error)
		ConflictResolveWorkflowExecution(ctx context.Context, request *persistence.ConflictResolveWorkflowExecutionRequest) (*persistence.ConflictResolveWorkflowExecutionResponse, error)
		SetWorkflowExecution(ctx context.Context, request *persistence.SetWorkflowExecutionRequest) (*persistence.SetWorkflowExecutionResponse, error)
		GetCurrentExecution(ctx context.Context, request *persistence.GetCurrentExecutionRequest) (*persistence.GetCurrentExecutionResponse, error)
		GetWorkflowExecution(ctx context.Context, request *persistence.GetWorkflowExecutionRequest) (*persistence.GetWorkflowExecutionResponse, error)
		// DeleteWorkflowExecution add task to delete visibility, current workflow execution, and deletes workflow execution.
		// If branchToken != nil, then delete history also, otherwise leave history.
		DeleteWorkflowExecution(ctx context.Context, workflowKey definition.WorkflowKey, branchToken []byte, closeExecutionVisibilityTaskID int64, workflowCloseTime time.Time, stage *tasks.DeleteWorkflowExecutionStage) error

		UnloadForOwnershipLost()

		StateMachineRegistry() *hsm.Registry
	}

	// A ControllableContext is a Context plus other methods needed by
	// the Controller.
	ControllableContext interface {
		Context
		pingable.Pingable

		IsValid() bool
		FinishStop()
	}
)
