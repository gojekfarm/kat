package pkg

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestPartition_ReassignPartitions_CreateTopicsToMoveFailure(t *testing.T) {
	executor := &MockExecutor{}
	io := &MockIo{}
	partition := &Partition{
		zookeeper:                  "zoo",
		executor:                   executor,
		io:                         io,
		kafkaPartitionReassignment: kafkaPartitionReassignment{},
	}
	topics := []string{"test-1", "test-2"}
	expectedErr := errors.New("error")
	io.On("WriteFile", mock.Anything, mock.Anything).Return(expectedErr)

	err := partition.ReassignPartitions(topics, "broker-list", 2, 10, 1, 100000)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	executor.AssertExpectations(t)
	io.AssertExpectations(t)
}

func TestPartition_ReassignPartitions_CreateTopicsSuccess_GenerateReassignmentFailure(t *testing.T) {
	executor := &MockExecutor{}
	io := &MockIo{}
	partition := &Partition{
		zookeeper:                  "zoo",
		executor:                   executor,
		io:                         io,
		kafkaPartitionReassignment: kafkaPartitionReassignment{},
	}
	topics := []string{"test-1", "test-2"}
	expectedTopicsToMove := topicsToMove{Topics: []map[string]string{{"topic": "test-1"}, {"topic": "test-2"}}}
	expectedErr := errors.New("error")
	io.On("WriteFile", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		expectedJson, _ := json.MarshalIndent(expectedTopicsToMove, "", "")
		assert.Equal(t, string(expectedJson), args[1])
	}).Return(nil)
	executor.On("Execute", mock.Anything, mock.Anything).Return(bytes.Buffer{}, expectedErr)

	err := partition.ReassignPartitions(topics, "broker-list", 2, 10, 1, 100000)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	executor.AssertExpectations(t)
	io.AssertExpectations(t)
}

func TestPartition_ReassignPartitions_GenerateReassignmentAndRollbackSuccess_ExecuteFailure(t *testing.T) {
	executor := &MockExecutor{}
	io := &MockIo{}
	partition := &Partition{
		zookeeper: "zoo",
		executor:  executor,
		io:        io,
		kafkaPartitionReassignment: kafkaPartitionReassignment{
			topicsToMoveJsonFile: "/tmp/topics-to-move-%d.json",
			reassignmentJsonFile: "/tmp/reassignment-%d.json",
			rollbackJsonFile:     "/tmp/rollback-%d.json",
		},
	}
	topics := []string{"test-1", "test-2"}

	expectedTopicsToMove := topicsToMove{Topics: []map[string]string{{"topic": "test-1"}, {"topic": "test-2"}}}
	expectedTopicsJson, _ := json.MarshalIndent(expectedTopicsToMove, "", "")
	expectedErr := errors.New("error")
	io.On("WriteFile", "/tmp/topics-to-move-0.json", string(expectedTopicsJson)).Return(nil)

	expectedFullReassignmentBytes := bytes.Buffer{}
	expectedFullReassignmentBytes.WriteString("Current partition replica assignment\n" +
		"{\"version\":1,\"partitions\":[{\"topic\":\"test-1\",\"partition\":0,\"replicas\":[6,1,2],\"log_dirs\":[\"any\",\"any\",\"any\"]}, {\"topic\":\"test-2\",\"partition\":0,\"replicas\":[4,2,5],\"log_dirs\":[\"any\",\"any\",\"any\"]}]}\n" +
		"                       \n" +
		"Proposed partition reassignment configuration\n" +
		"{\"version\":1,\"partitions\":[{\"topic\":\"test-1\",\"partition\":0,\"replicas\":[1,2,3],\"log_dirs\":[\"any\",\"any\",\"any\"]}, {\"topic\":\"test-2\",\"partition\":0,\"replicas\":[3,5,6],\"log_dirs\":[\"any\",\"any\",\"any\"]}]}\n")
	executor.On("Execute", "kafka-reassign-partitions", []string{"--zookeeper", "zoo", "--broker-list", "broker-list", "--topics-to-move-json-file", "/tmp/topics-to-move-0.json", "--generate"}).Return(expectedFullReassignmentBytes, nil)

	expectedRollbackJson := "{\"version\":1,\"partitions\":[{\"topic\":\"test-1\",\"partition\":0,\"replicas\":[6,1,2],\"log_dirs\":[\"any\",\"any\",\"any\"]}, {\"topic\":\"test-2\",\"partition\":0,\"replicas\":[4,2,5],\"log_dirs\":[\"any\",\"any\",\"any\"]}]}"
	expectedReassignmentJson := "{\"version\":1,\"partitions\":[{\"topic\":\"test-1\",\"partition\":0,\"replicas\":[1,2,3],\"log_dirs\":[\"any\",\"any\",\"any\"]}, {\"topic\":\"test-2\",\"partition\":0,\"replicas\":[3,5,6],\"log_dirs\":[\"any\",\"any\",\"any\"]}]}"
	io.On("WriteFile", "/tmp/rollback-0.json", expectedRollbackJson).Return(nil)
	io.On("WriteFile", "/tmp/reassignment-0.json", expectedReassignmentJson).Return(nil)

	executor.On("Execute", "kafka-reassign-partitions", []string{"--zookeeper", "zoo", "--reassignment-json-file", "/tmp/reassignment-0.json", "--throttle", "100000", "--execute"}).Return(bytes.Buffer{}, expectedErr)

	err := partition.ReassignPartitions(topics, "broker-list", 2, 10, 1, 100000)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	executor.AssertExpectations(t)
	io.AssertExpectations(t)
}

func TestPartition_ReassignPartitions_ExecuteSuccess_PollFailure(t *testing.T) {
	executor := &MockExecutor{}
	io := &MockIo{}
	partition := &Partition{
		zookeeper: "zoo",
		executor:  executor,
		io:        io,
		kafkaPartitionReassignment: kafkaPartitionReassignment{
			topicsToMoveJsonFile: "/tmp/topics-to-move-%d.json",
			reassignmentJsonFile: "/tmp/reassignment-%d.json",
			rollbackJsonFile:     "/tmp/rollback-%d.json",
		},
	}
	topics := []string{"test-1", "test-2"}

	expectedTopicsToMove := topicsToMove{Topics: []map[string]string{{"topic": "test-1"}, {"topic": "test-2"}}}
	expectedTopicsJson, _ := json.MarshalIndent(expectedTopicsToMove, "", "")
	expectedErr := errors.New("Partition Reassignment failed: Reassignment of partition test-1-0 failed")
	io.On("WriteFile", "/tmp/topics-to-move-0.json", string(expectedTopicsJson)).Return(nil)

	expectedFullReassignmentBytes := bytes.Buffer{}
	expectedFullReassignmentBytes.WriteString("Current partition replica assignment\n" +
		"{\"version\":1,\"partitions\":[{\"topic\":\"test-1\",\"partition\":0,\"replicas\":[6,1,2],\"log_dirs\":[\"any\",\"any\",\"any\"]}, {\"topic\":\"test-2\",\"partition\":0,\"replicas\":[4,2,5],\"log_dirs\":[\"any\",\"any\",\"any\"]}]}\n" +
		"                       \n" +
		"Proposed partition reassignment configuration\n" +
		"{\"version\":1,\"partitions\":[{\"topic\":\"test-1\",\"partition\":0,\"replicas\":[1,2,3],\"log_dirs\":[\"any\",\"any\",\"any\"]}, {\"topic\":\"test-2\",\"partition\":0,\"replicas\":[3,5,6],\"log_dirs\":[\"any\",\"any\",\"any\"]}]}\n")
	executor.On("Execute", "kafka-reassign-partitions", []string{"--zookeeper", "zoo", "--broker-list", "broker-list", "--topics-to-move-json-file", "/tmp/topics-to-move-0.json", "--generate"}).Return(expectedFullReassignmentBytes, nil)

	expectedRollbackJson := "{\"version\":1,\"partitions\":[{\"topic\":\"test-1\",\"partition\":0,\"replicas\":[6,1,2],\"log_dirs\":[\"any\",\"any\",\"any\"]}, {\"topic\":\"test-2\",\"partition\":0,\"replicas\":[4,2,5],\"log_dirs\":[\"any\",\"any\",\"any\"]}]}"
	expectedReassignmentJson := "{\"version\":1,\"partitions\":[{\"topic\":\"test-1\",\"partition\":0,\"replicas\":[1,2,3],\"log_dirs\":[\"any\",\"any\",\"any\"]}, {\"topic\":\"test-2\",\"partition\":0,\"replicas\":[3,5,6],\"log_dirs\":[\"any\",\"any\",\"any\"]}]}"
	io.On("WriteFile", "/tmp/rollback-0.json", expectedRollbackJson).Return(nil)
	io.On("WriteFile", "/tmp/reassignment-0.json", expectedReassignmentJson).Return(nil)

	executor.On("Execute", "kafka-reassign-partitions", []string{"--zookeeper", "zoo", "--reassignment-json-file", "/tmp/reassignment-0.json", "--throttle", "100000", "--execute"}).Return(bytes.Buffer{}, nil)

	expectedVerificationBytes := bytes.Buffer{}
	expectedVerificationBytes.WriteString("Status of partition reassignment: \n" +
		"Reassignment of partition test-1-0 failed\n" +
		"Reassignment of partition test-2-0 completed successfully\n")
	executor.On("Execute", "kafka-reassign-partitions", []string{"--zookeeper", "zoo", "--reassignment-json-file", "/tmp/reassignment-0.json", "--verify"}).Return(expectedVerificationBytes, nil)

	err := partition.ReassignPartitions(topics, "broker-list", 2, 1, 1, 100000)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	executor.AssertExpectations(t)
	io.AssertExpectations(t)
}

func TestPartition_ReassignPartitions_Success(t *testing.T) {
	executor := &MockExecutor{}
	io := &MockIo{}
	partition := &Partition{
		zookeeper: "zoo",
		executor:  executor,
		io:        io,
		kafkaPartitionReassignment: kafkaPartitionReassignment{
			topicsToMoveJsonFile: "/tmp/topics-to-move-%d.json",
			reassignmentJsonFile: "/tmp/reassignment-%d.json",
			rollbackJsonFile:     "/tmp/rollback-%d.json",
		},
	}
	topics := []string{"test-1", "test-2"}

	expectedTopicsToMove := topicsToMove{Topics: []map[string]string{{"topic": "test-1"}, {"topic": "test-2"}}}
	expectedTopicsJson, _ := json.MarshalIndent(expectedTopicsToMove, "", "")
	io.On("WriteFile", "/tmp/topics-to-move-0.json", string(expectedTopicsJson)).Return(nil)

	expectedFullReassignmentBytes := bytes.Buffer{}
	expectedFullReassignmentBytes.WriteString("Current partition replica assignment\n" +
		"{\"version\":1,\"partitions\":[{\"topic\":\"test-1\",\"partition\":0,\"replicas\":[6,1,2],\"log_dirs\":[\"any\",\"any\",\"any\"]}, {\"topic\":\"test-2\",\"partition\":0,\"replicas\":[4,2,5],\"log_dirs\":[\"any\",\"any\",\"any\"]}]}\n" +
		"                       \n" +
		"Proposed partition reassignment configuration\n" +
		"{\"version\":1,\"partitions\":[{\"topic\":\"test-1\",\"partition\":0,\"replicas\":[1,2,3],\"log_dirs\":[\"any\",\"any\",\"any\"]}, {\"topic\":\"test-2\",\"partition\":0,\"replicas\":[3,5,6],\"log_dirs\":[\"any\",\"any\",\"any\"]}]}\n")
	executor.On("Execute", "kafka-reassign-partitions", []string{"--zookeeper", "zoo", "--broker-list", "broker-list", "--topics-to-move-json-file", "/tmp/topics-to-move-0.json", "--generate"}).Return(expectedFullReassignmentBytes, nil)

	expectedRollbackJson := "{\"version\":1,\"partitions\":[{\"topic\":\"test-1\",\"partition\":0,\"replicas\":[6,1,2],\"log_dirs\":[\"any\",\"any\",\"any\"]}, {\"topic\":\"test-2\",\"partition\":0,\"replicas\":[4,2,5],\"log_dirs\":[\"any\",\"any\",\"any\"]}]}"
	expectedReassignmentJson := "{\"version\":1,\"partitions\":[{\"topic\":\"test-1\",\"partition\":0,\"replicas\":[1,2,3],\"log_dirs\":[\"any\",\"any\",\"any\"]}, {\"topic\":\"test-2\",\"partition\":0,\"replicas\":[3,5,6],\"log_dirs\":[\"any\",\"any\",\"any\"]}]}"
	io.On("WriteFile", "/tmp/rollback-0.json", expectedRollbackJson).Return(nil)
	io.On("WriteFile", "/tmp/reassignment-0.json", expectedReassignmentJson).Return(nil)

	executor.On("Execute", "kafka-reassign-partitions", []string{"--zookeeper", "zoo", "--reassignment-json-file", "/tmp/reassignment-0.json", "--throttle", "100000", "--execute"}).Return(bytes.Buffer{}, nil)

	expectedVerificationBytes := bytes.Buffer{}
	expectedVerificationBytes.WriteString("Status of partition reassignment: \n" +
		"Reassignment of partition test-1-0 completed successfully\n" +
		"Reassignment of partition test-2-0 completed successfully\n")
	executor.On("Execute", "kafka-reassign-partitions", []string{"--zookeeper", "zoo", "--reassignment-json-file", "/tmp/reassignment-0.json", "--verify"}).Return(expectedVerificationBytes, nil)

	err := partition.ReassignPartitions(topics, "broker-list", 2, 1, 1, 100000)
	assert.NoError(t, err)
	executor.AssertExpectations(t)
	io.AssertExpectations(t)
}

func TestPartition_ReassignPartitions_PollUntilTimeoutIfNotYetSuccessful(t *testing.T) {
	executor := &MockExecutor{}
	io := &MockIo{}
	partition := &Partition{
		zookeeper: "zoo",
		executor:  executor,
		io:        io,
		kafkaPartitionReassignment: kafkaPartitionReassignment{
			topicsToMoveJsonFile: "/tmp/topics-to-move-%d.json",
			reassignmentJsonFile: "/tmp/reassignment-%d.json",
			rollbackJsonFile:     "/tmp/rollback-%d.json",
		},
	}
	topics := []string{"test-1", "test-2"}
	expectedErr := errors.New("Partition Reassignment failed: Reassignment of partition test-1-0 is inprogress")

	expectedTopicsToMove := topicsToMove{Topics: []map[string]string{{"topic": "test-1"}, {"topic": "test-2"}}}
	expectedTopicsJson, _ := json.MarshalIndent(expectedTopicsToMove, "", "")
	io.On("WriteFile", "/tmp/topics-to-move-0.json", string(expectedTopicsJson)).Return(nil)

	expectedFullReassignmentBytes := bytes.Buffer{}
	expectedFullReassignmentBytes.WriteString("Current partition replica assignment\n" +
		"{\"version\":1,\"partitions\":[{\"topic\":\"test-1\",\"partition\":0,\"replicas\":[6,1,2],\"log_dirs\":[\"any\",\"any\",\"any\"]}, {\"topic\":\"test-2\",\"partition\":0,\"replicas\":[4,2,5],\"log_dirs\":[\"any\",\"any\",\"any\"]}]}\n" +
		"                       \n" +
		"Proposed partition reassignment configuration\n" +
		"{\"version\":1,\"partitions\":[{\"topic\":\"test-1\",\"partition\":0,\"replicas\":[1,2,3],\"log_dirs\":[\"any\",\"any\",\"any\"]}, {\"topic\":\"test-2\",\"partition\":0,\"replicas\":[3,5,6],\"log_dirs\":[\"any\",\"any\",\"any\"]}]}\n")
	executor.On("Execute", "kafka-reassign-partitions", []string{"--zookeeper", "zoo", "--broker-list", "broker-list", "--topics-to-move-json-file", "/tmp/topics-to-move-0.json", "--generate"}).Return(expectedFullReassignmentBytes, nil)

	expectedRollbackJson := "{\"version\":1,\"partitions\":[{\"topic\":\"test-1\",\"partition\":0,\"replicas\":[6,1,2],\"log_dirs\":[\"any\",\"any\",\"any\"]}, {\"topic\":\"test-2\",\"partition\":0,\"replicas\":[4,2,5],\"log_dirs\":[\"any\",\"any\",\"any\"]}]}"
	expectedReassignmentJson := "{\"version\":1,\"partitions\":[{\"topic\":\"test-1\",\"partition\":0,\"replicas\":[1,2,3],\"log_dirs\":[\"any\",\"any\",\"any\"]}, {\"topic\":\"test-2\",\"partition\":0,\"replicas\":[3,5,6],\"log_dirs\":[\"any\",\"any\",\"any\"]}]}"
	io.On("WriteFile", "/tmp/rollback-0.json", expectedRollbackJson).Return(nil)
	io.On("WriteFile", "/tmp/reassignment-0.json", expectedReassignmentJson).Return(nil)

	executor.On("Execute", "kafka-reassign-partitions", []string{"--zookeeper", "zoo", "--reassignment-json-file", "/tmp/reassignment-0.json", "--throttle", "100000", "--execute"}).Return(bytes.Buffer{}, nil)

	expectedVerificationBytes := bytes.Buffer{}
	expectedVerificationBytes.WriteString("Status of partition reassignment: \n" +
		"Reassignment of partition test-1-0 is inprogress\n" +
		"Reassignment of partition test-2-0 completed successfully\n")
	executor.On("Execute", "kafka-reassign-partitions", []string{"--zookeeper", "zoo", "--reassignment-json-file", "/tmp/reassignment-0.json", "--verify"}).Return(expectedVerificationBytes, nil).Times(3)

	err := partition.ReassignPartitions(topics, "broker-list", 2, 3, 1, 100000)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	executor.AssertExpectations(t)
	io.AssertExpectations(t)
}

func TestPartition_ReassignPartitions_Success_ForMultipleBatches(t *testing.T) {
	executor := &MockExecutor{}
	io := &MockIo{}
	partition := &Partition{
		zookeeper: "zoo",
		executor:  executor,
		io:        io,
		kafkaPartitionReassignment: kafkaPartitionReassignment{
			topicsToMoveJsonFile: "/tmp/topics-to-move-%d.json",
			reassignmentJsonFile: "/tmp/reassignment-%d.json",
			rollbackJsonFile:     "/tmp/rollback-%d.json",
		},
	}
	topics := []string{"test-1", "test-2"}

	expectedTopicsToMove1 := topicsToMove{Topics: []map[string]string{{"topic": "test-1"}}}
	expectedTopicsJson1, _ := json.MarshalIndent(expectedTopicsToMove1, "", "")
	io.On("WriteFile", "/tmp/topics-to-move-0.json", string(expectedTopicsJson1)).Return(nil)

	expectedTopicsToMove2 := topicsToMove{Topics: []map[string]string{{"topic": "test-2"}}}
	expectedTopicsJson2, _ := json.MarshalIndent(expectedTopicsToMove2, "", "")
	io.On("WriteFile", "/tmp/topics-to-move-1.json", string(expectedTopicsJson2)).Return(nil)

	expectedFullReassignmentBytes1 := bytes.Buffer{}
	expectedFullReassignmentBytes1.WriteString("Current partition replica assignment\n" +
		"{\"version\":1,\"partitions\":[{\"topic\":\"test-1\",\"partition\":0,\"replicas\":[6,1,2],\"log_dirs\":[\"any\",\"any\",\"any\"]}]}\n" +
		"                       \n" +
		"Proposed partition reassignment configuration\n" +
		"{\"version\":1,\"partitions\":[{\"topic\":\"test-1\",\"partition\":0,\"replicas\":[1,2,3],\"log_dirs\":[\"any\",\"any\",\"any\"]}]}\n")
	executor.On("Execute", "kafka-reassign-partitions", []string{"--zookeeper", "zoo", "--broker-list", "broker-list", "--topics-to-move-json-file", "/tmp/topics-to-move-0.json", "--generate"}).Return(expectedFullReassignmentBytes1, nil)

	expectedRollbackJson1 := "{\"version\":1,\"partitions\":[{\"topic\":\"test-1\",\"partition\":0,\"replicas\":[6,1,2],\"log_dirs\":[\"any\",\"any\",\"any\"]}]}"
	expectedReassignmentJson1 := "{\"version\":1,\"partitions\":[{\"topic\":\"test-1\",\"partition\":0,\"replicas\":[1,2,3],\"log_dirs\":[\"any\",\"any\",\"any\"]}]}"
	io.On("WriteFile", "/tmp/rollback-0.json", expectedRollbackJson1).Return(nil)
	io.On("WriteFile", "/tmp/reassignment-0.json", expectedReassignmentJson1).Return(nil)

	expectedFullReassignmentBytes2 := bytes.Buffer{}
	expectedFullReassignmentBytes2.WriteString("Current partition replica assignment\n" +
		"{\"version\":1,\"partitions\":[{\"topic\":\"test-2\",\"partition\":0,\"replicas\":[4,2,5],\"log_dirs\":[\"any\",\"any\",\"any\"]}]}\n" +
		"                       \n" +
		"Proposed partition reassignment configuration\n" +
		"{\"version\":1,\"partitions\":[{\"topic\":\"test-2\",\"partition\":0,\"replicas\":[3,5,6],\"log_dirs\":[\"any\",\"any\",\"any\"]}]}\n")
	executor.On("Execute", "kafka-reassign-partitions", []string{"--zookeeper", "zoo", "--broker-list", "broker-list", "--topics-to-move-json-file", "/tmp/topics-to-move-1.json", "--generate"}).Return(expectedFullReassignmentBytes2, nil)

	expectedRollbackJson2 := "{\"version\":1,\"partitions\":[{\"topic\":\"test-2\",\"partition\":0,\"replicas\":[4,2,5],\"log_dirs\":[\"any\",\"any\",\"any\"]}]}"
	expectedReassignmentJson2 := "{\"version\":1,\"partitions\":[{\"topic\":\"test-2\",\"partition\":0,\"replicas\":[3,5,6],\"log_dirs\":[\"any\",\"any\",\"any\"]}]}"
	io.On("WriteFile", "/tmp/rollback-1.json", expectedRollbackJson2).Return(nil)
	io.On("WriteFile", "/tmp/reassignment-1.json", expectedReassignmentJson2).Return(nil)

	executor.On("Execute", "kafka-reassign-partitions", []string{"--zookeeper", "zoo", "--reassignment-json-file", "/tmp/reassignment-0.json", "--throttle", "100000", "--execute"}).Return(bytes.Buffer{}, nil)
	executor.On("Execute", "kafka-reassign-partitions", []string{"--zookeeper", "zoo", "--reassignment-json-file", "/tmp/reassignment-1.json", "--throttle", "100000", "--execute"}).Return(bytes.Buffer{}, nil)

	expectedVerificationBytes1 := bytes.Buffer{}
	expectedVerificationBytes1.WriteString("Status of partition reassignment: \n" +
		"Reassignment of partition test-1-0 completed successfully\n")
	executor.On("Execute", "kafka-reassign-partitions", []string{"--zookeeper", "zoo", "--reassignment-json-file", "/tmp/reassignment-0.json", "--verify"}).Return(expectedVerificationBytes1, nil)

	expectedVerificationBytes2 := bytes.Buffer{}
	expectedVerificationBytes2.WriteString("Status of partition reassignment: \n" +
		"Reassignment of partition test-2-0 completed successfully\n")
	executor.On("Execute", "kafka-reassign-partitions", []string{"--zookeeper", "zoo", "--reassignment-json-file", "/tmp/reassignment-1.json", "--verify"}).Return(expectedVerificationBytes2, nil)

	err := partition.ReassignPartitions(topics, "broker-list", 1, 1, 1, 100000)
	assert.NoError(t, err)
	executor.AssertExpectations(t)
	io.AssertExpectations(t)
}

func TestPartition_IncreaseReplication_WriteReassignmentFailure(t *testing.T) {
	executor := &MockExecutor{}
	io := &MockIo{}
	partition := &Partition{
		zookeeper: "zoo",
		executor:  executor,
		io:        io,
		kafkaPartitionReassignment: kafkaPartitionReassignment{
			topicsToMoveJsonFile: "/tmp/topics-to-move-%d.json",
			reassignmentJsonFile: "/tmp/reassignment-%d.json",
			rollbackJsonFile:     "/tmp/rollback-%d.json",
		},
	}
	expectedErr := errors.New("error")

	topicsMetadata := []*TopicMetadata{{
		Err:        nil,
		Name:       "test-1",
		IsInternal: false,
		Partitions: []*PartitionMetadata{{
			Err:             nil,
			ID:              1,
			Leader:          1,
			Replicas:        []int32{1},
			Isr:             []int32{1},
			OfflineReplicas: nil,
		}},
	}}
	io.On("WriteFile", "/tmp/reassignment-0.json", mock.Anything).Return(expectedErr)

	err := partition.IncreaseReplication(topicsMetadata, 1, 1, 1, 3, 1, 100000)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	executor.AssertExpectations(t)
	io.AssertExpectations(t)
}

func TestPartition_IncreaseReplication_WriteReassignmentSuccess_ExecuteFailure(t *testing.T) {
	executor := &MockExecutor{}
	io := &MockIo{}
	partition := &Partition{
		zookeeper: "zoo",
		executor:  executor,
		io:        io,
		kafkaPartitionReassignment: kafkaPartitionReassignment{
			topicsToMoveJsonFile: "/tmp/topics-to-move-%d.json",
			reassignmentJsonFile: "/tmp/reassignment-%d.json",
			rollbackJsonFile:     "/tmp/rollback-%d.json",
		},
	}
	expectedErr := errors.New("error")

	topicsMetadata := []*TopicMetadata{{
		Err:        nil,
		Name:       "test-1",
		IsInternal: false,
		Partitions: []*PartitionMetadata{{
			Err:             nil,
			ID:              1,
			Leader:          1,
			Replicas:        []int32{1},
			Isr:             []int32{1},
			OfflineReplicas: nil,
		}},
	}}
	io.On("WriteFile", "/tmp/reassignment-0.json", mock.Anything).Return(nil)
	executor.On("Execute", "kafka-reassign-partitions", []string{"--zookeeper", "zoo", "--reassignment-json-file", "/tmp/reassignment-0.json", "--throttle", "100000", "--execute"}).Return(bytes.Buffer{}, expectedErr)

	err := partition.IncreaseReplication(topicsMetadata, 1, 1, 1, 3, 1, 100000)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	executor.AssertExpectations(t)
	io.AssertExpectations(t)
}

func TestPartition_IncreaseReplication_ExecuteSuccess_RollbackJsonFailure(t *testing.T) {
	executor := &MockExecutor{}
	io := &MockIo{}
	partition := &Partition{
		zookeeper: "zoo",
		executor:  executor,
		io:        io,
		kafkaPartitionReassignment: kafkaPartitionReassignment{
			topicsToMoveJsonFile: "/tmp/topics-to-move-%d.json",
			reassignmentJsonFile: "/tmp/reassignment-%d.json",
			rollbackJsonFile:     "/tmp/rollback-%d.json",
		},
	}
	expectedErr := errors.New("error")

	topicsMetadata := []*TopicMetadata{{
		Err:        nil,
		Name:       "test-1",
		IsInternal: false,
		Partitions: []*PartitionMetadata{{
			Err:             nil,
			ID:              1,
			Leader:          1,
			Replicas:        []int32{1},
			Isr:             []int32{1},
			OfflineReplicas: nil,
		}},
	}}
	io.On("WriteFile", "/tmp/reassignment-0.json", mock.Anything).Return(nil)
	io.On("WriteFile", "/tmp/rollback-0.json", mock.Anything).Return(expectedErr)

	expectedFullReassignmentBytes := bytes.Buffer{}
	expectedFullReassignmentBytes.WriteString("Current partition replica assignment\n" + "\n" +
		"{\"version\":1,\"partitions\":[{\"topic\":\"test-1\",\"partition\":0,\"replicas\":[6,1,2],\"log_dirs\":[\"any\",\"any\",\"any\"]}, {\"topic\":\"test-2\",\"partition\":0,\"replicas\":[4,2,5],\"log_dirs\":[\"any\",\"any\",\"any\"]}]}\n")
	executor.On("Execute", "kafka-reassign-partitions", []string{"--zookeeper", "zoo", "--reassignment-json-file", "/tmp/reassignment-0.json", "--throttle", "100000", "--execute"}).Return(expectedFullReassignmentBytes, nil)

	err := partition.IncreaseReplication(topicsMetadata, 1, 1, 1, 3, 1, 100000)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	executor.AssertExpectations(t)
	io.AssertExpectations(t)
}

func TestPartition_IncreaseReplication_RollbackJsonSuccess_PollFailure(t *testing.T) {
	executor := &MockExecutor{}
	io := &MockIo{}
	partition := &Partition{
		zookeeper: "zoo",
		executor:  executor,
		io:        io,
		kafkaPartitionReassignment: kafkaPartitionReassignment{
			topicsToMoveJsonFile: "/tmp/topics-to-move-%d.json",
			reassignmentJsonFile: "/tmp/reassignment-%d.json",
			rollbackJsonFile:     "/tmp/rollback-%d.json",
		},
	}
	expectedErr := errors.New("Partition Reassignment failed: Reassignment of partition test-1-0 failed")

	topicsMetadata := []*TopicMetadata{{
		Err:        nil,
		Name:       "test-1",
		IsInternal: false,
		Partitions: []*PartitionMetadata{{
			Err:             nil,
			ID:              1,
			Leader:          1,
			Replicas:        []int32{1},
			Isr:             []int32{1},
			OfflineReplicas: nil,
		}},
	}}
	io.On("WriteFile", "/tmp/reassignment-0.json", mock.Anything).Return(nil)
	io.On("WriteFile", "/tmp/rollback-0.json", mock.Anything).Return(nil)

	expectedFullReassignmentBytes := bytes.Buffer{}
	expectedFullReassignmentBytes.WriteString("Current partition replica assignment\n" + "\n" +
		"{\"version\":1,\"partitions\":[{\"topic\":\"test-1\",\"partition\":0,\"replicas\":[6,1,2],\"log_dirs\":[\"any\",\"any\",\"any\"]}, {\"topic\":\"test-2\",\"partition\":0,\"replicas\":[4,2,5],\"log_dirs\":[\"any\",\"any\",\"any\"]}]}\n")
	executor.On("Execute", "kafka-reassign-partitions", []string{"--zookeeper", "zoo", "--reassignment-json-file", "/tmp/reassignment-0.json", "--throttle", "100000", "--execute"}).Return(expectedFullReassignmentBytes, nil)

	expectedVerificationBytes := bytes.Buffer{}
	expectedVerificationBytes.WriteString("Status of partition reassignment: \n" +
		"Reassignment of partition test-1-0 failed\n")
	executor.On("Execute", "kafka-reassign-partitions", []string{"--zookeeper", "zoo", "--reassignment-json-file", "/tmp/reassignment-0.json", "--verify"}).Return(expectedVerificationBytes, nil)

	err := partition.IncreaseReplication(topicsMetadata, 1, 1, 1, 1, 1, 100000)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	executor.AssertExpectations(t)
	io.AssertExpectations(t)
}

func TestPartition_IncreaseReplicationSuccess(t *testing.T) {
	executor := &MockExecutor{}
	io := &MockIo{}
	partition := &Partition{
		zookeeper: "zoo",
		executor:  executor,
		io:        io,
		kafkaPartitionReassignment: kafkaPartitionReassignment{
			topicsToMoveJsonFile: "/tmp/topics-to-move-%d.json",
			reassignmentJsonFile: "/tmp/reassignment-%d.json",
			rollbackJsonFile:     "/tmp/rollback-%d.json",
		},
	}
	topicsMetadata := []*TopicMetadata{{
		Err:        nil,
		Name:       "test-1",
		IsInternal: false,
		Partitions: []*PartitionMetadata{{
			Err:             nil,
			ID:              1,
			Leader:          1,
			Replicas:        []int32{1},
			Isr:             []int32{1},
			OfflineReplicas: nil,
		}},
	}}
	io.On("WriteFile", "/tmp/reassignment-0.json", mock.Anything).Return(nil)
	io.On("WriteFile", "/tmp/rollback-0.json", mock.Anything).Return(nil)

	expectedFullReassignmentBytes := bytes.Buffer{}
	expectedFullReassignmentBytes.WriteString("Current partition replica assignment\n" + "\n" +
		"{\"version\":1,\"partitions\":[{\"topic\":\"test-1\",\"partition\":0,\"replicas\":[6,1,2],\"log_dirs\":[\"any\",\"any\",\"any\"]}, {\"topic\":\"test-2\",\"partition\":0,\"replicas\":[4,2,5],\"log_dirs\":[\"any\",\"any\",\"any\"]}]}\n")
	executor.On("Execute", "kafka-reassign-partitions", []string{"--zookeeper", "zoo", "--reassignment-json-file", "/tmp/reassignment-0.json", "--throttle", "100000", "--execute"}).Return(expectedFullReassignmentBytes, nil)

	expectedVerificationBytes := bytes.Buffer{}
	expectedVerificationBytes.WriteString("Status of partition reassignment: \n" +
		"Reassignment of partition test-1-0 completed successfully\n")
	executor.On("Execute", "kafka-reassign-partitions", []string{"--zookeeper", "zoo", "--reassignment-json-file", "/tmp/reassignment-0.json", "--verify"}).Return(expectedVerificationBytes, nil)

	err := partition.IncreaseReplication(topicsMetadata, 1, 1, 1, 1, 1, 100000)
	assert.NoError(t, err)
	executor.AssertExpectations(t)
	io.AssertExpectations(t)
}

func TestPartition_IncreaseReplication__PollUntilTimeoutIfNotYetSuccessful(t *testing.T) {
	executor := &MockExecutor{}
	io := &MockIo{}
	partition := &Partition{
		zookeeper: "zoo",
		executor:  executor,
		io:        io,
		kafkaPartitionReassignment: kafkaPartitionReassignment{
			topicsToMoveJsonFile: "/tmp/topics-to-move-%d.json",
			reassignmentJsonFile: "/tmp/reassignment-%d.json",
			rollbackJsonFile:     "/tmp/rollback-%d.json",
		},
	}
	expectedErr := errors.New("Partition Reassignment failed: Reassignment of partition test-1-0 is inprogress")
	topicsMetadata := []*TopicMetadata{{
		Err:        nil,
		Name:       "test-1",
		IsInternal: false,
		Partitions: []*PartitionMetadata{{
			Err:             nil,
			ID:              1,
			Leader:          1,
			Replicas:        []int32{1},
			Isr:             []int32{1},
			OfflineReplicas: nil,
		}},
	}}
	io.On("WriteFile", "/tmp/reassignment-0.json", mock.Anything).Return(nil)
	io.On("WriteFile", "/tmp/rollback-0.json", mock.Anything).Return(nil)

	expectedFullReassignmentBytes := bytes.Buffer{}
	expectedFullReassignmentBytes.WriteString("Current partition replica assignment\n" + "\n" +
		"{\"version\":1,\"partitions\":[{\"topic\":\"test-1\",\"partition\":0,\"replicas\":[6,1,2],\"log_dirs\":[\"any\",\"any\",\"any\"]}, {\"topic\":\"test-2\",\"partition\":0,\"replicas\":[4,2,5],\"log_dirs\":[\"any\",\"any\",\"any\"]}]}\n")
	executor.On("Execute", "kafka-reassign-partitions", []string{"--zookeeper", "zoo", "--reassignment-json-file", "/tmp/reassignment-0.json", "--throttle", "100000", "--execute"}).Return(expectedFullReassignmentBytes, nil)

	expectedVerificationBytes := bytes.Buffer{}
	expectedVerificationBytes.WriteString("Status of partition reassignment: \n" +
		"Reassignment of partition test-1-0 is inprogress\n")
	executor.On("Execute", "kafka-reassign-partitions", []string{"--zookeeper", "zoo", "--reassignment-json-file", "/tmp/reassignment-0.json", "--verify"}).Return(expectedVerificationBytes, nil).Times(3)

	err := partition.IncreaseReplication(topicsMetadata, 1, 1, 1, 3, 1, 100000)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	executor.AssertExpectations(t)
	io.AssertExpectations(t)
}

func TestBuildReassignmentJson(suite *testing.T) {
	suite.Run("Build Reassignment Json", func(t *testing.T) {
		partitionMetadata1 := PartitionMetadata{ID: 8, Leader: 6, Replicas: []int32{6}}
		partitionMetadata2 := PartitionMetadata{ID: 11, Leader: 3, Replicas: []int32{3}}
		partitionMetadata3 := PartitionMetadata{ID: 2, Leader: 6, Replicas: []int32{6}}
		partitionMetadata4 := PartitionMetadata{ID: 5, Leader: 3, Replicas: []int32{3}}
		partitionMetadata5 := PartitionMetadata{ID: 4, Leader: 2, Replicas: []int32{2}}
		partitionMetadata6 := PartitionMetadata{ID: 7, Leader: 5, Replicas: []int32{5}}
		partitionMetadata7 := PartitionMetadata{ID: 10, Leader: 2, Replicas: []int32{2}}
		partitionMetadata8 := PartitionMetadata{ID: 1, Leader: 5, Replicas: []int32{5}}
		partitionMetadata9 := PartitionMetadata{ID: 9, Leader: 1, Replicas: []int32{1}}
		partitionMetadata10 := PartitionMetadata{ID: 3, Leader: 1, Replicas: []int32{1}}
		partitionMetadata11 := PartitionMetadata{ID: 6, Leader: 4, Replicas: []int32{4}}
		partitionMetadata12 := PartitionMetadata{ID: 0, Leader: 4, Replicas: []int32{4}}
		topicMetadata := TopicMetadata{Name: "topic", Partitions: []*PartitionMetadata{&partitionMetadata1, &partitionMetadata2, &partitionMetadata3, &partitionMetadata4, &partitionMetadata5, &partitionMetadata6, &partitionMetadata7, &partitionMetadata8, &partitionMetadata9, &partitionMetadata10, &partitionMetadata11, &partitionMetadata12}}
		expectedJSONForReplicationFactor3 := reassignmentJSON{Version: 1, Partitions: []partitionDetail{{Topic: "topic", Partition: 8, Replicas: []int32{6, 1, 2}}, {Topic: "topic", Partition: 11, Replicas: []int32{3, 4, 5}}, {Topic: "topic", Partition: 2, Replicas: []int32{6, 3, 4}}, {Topic: "topic", Partition: 5, Replicas: []int32{3, 6, 1}}, {Topic: "topic", Partition: 4, Replicas: []int32{2, 3, 4}}, {Topic: "topic", Partition: 7, Replicas: []int32{5, 6, 1}}, {Topic: "topic", Partition: 10, Replicas: []int32{2, 5, 6}}, {Topic: "topic", Partition: 1, Replicas: []int32{5, 2, 3}}, {Topic: "topic", Partition: 9, Replicas: []int32{1, 2, 3}}, {Topic: "topic", Partition: 3, Replicas: []int32{1, 4, 5}}, {Topic: "topic", Partition: 6, Replicas: []int32{4, 5, 6}}, {Topic: "topic", Partition: 0, Replicas: []int32{4, 1, 2}}}}
		expectedJSONForReplicationFactor4 := reassignmentJSON{Version: 1, Partitions: []partitionDetail{{Topic: "topic", Partition: 8, Replicas: []int32{6, 1, 2, 3}}, {Topic: "topic", Partition: 11, Replicas: []int32{3, 4, 5, 6}}, {Topic: "topic", Partition: 2, Replicas: []int32{6, 4, 5, 1}}, {Topic: "topic", Partition: 5, Replicas: []int32{3, 1, 2, 4}}, {Topic: "topic", Partition: 4, Replicas: []int32{2, 3, 4, 5}}, {Topic: "topic", Partition: 7, Replicas: []int32{5, 6, 1, 2}}, {Topic: "topic", Partition: 10, Replicas: []int32{2, 6, 1, 3}}, {Topic: "topic", Partition: 1, Replicas: []int32{5, 3, 4, 6}}, {Topic: "topic", Partition: 9, Replicas: []int32{1, 2, 3, 4}}, {Topic: "topic", Partition: 3, Replicas: []int32{1, 5, 6, 2}}, {Topic: "topic", Partition: 6, Replicas: []int32{4, 5, 6, 1}}, {Topic: "topic", Partition: 0, Replicas: []int32{4, 2, 3, 5}}}}

		actualJSONForReplicationFactor3 := buildReassignmentJSON([]*TopicMetadata{&topicMetadata}, 3, 6)
		actualJSONForReplicationFactor4 := buildReassignmentJSON([]*TopicMetadata{&topicMetadata}, 4, 6)

		assert.Equal(t, expectedJSONForReplicationFactor3, actualJSONForReplicationFactor3)
		assert.Equal(t, expectedJSONForReplicationFactor4, actualJSONForReplicationFactor4)
	})
}