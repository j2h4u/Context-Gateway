package preemptive_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/compresr/context-gateway/internal/preemptive"
)

// =============================================================================
// WORKER TESTS
// =============================================================================

// TestWorker tests the basic worker functionality using the actual preemptive types.
// Note: These tests use the actual implementation interfaces.

func TestWorkerJob_StatusTypes(t *testing.T) {
	// Verify job status constants are defined correctly
	assert.Equal(t, preemptive.JobStatus("queued"), preemptive.JobStatusQueued)
	assert.Equal(t, preemptive.JobStatus("running"), preemptive.JobStatusRunning)
	assert.Equal(t, preemptive.JobStatus("completed"), preemptive.JobStatusCompleted)
	assert.Equal(t, preemptive.JobStatus("failed"), preemptive.JobStatusFailed)
	assert.Equal(t, preemptive.JobStatus("cancelled"), preemptive.JobStatusCancelled)
}

func TestSummarizationJob_Structure(t *testing.T) {
	// Test that SummarizationJob has the expected fields
	now := time.Now()
	job := &preemptive.SummarizationJob{
		ID:           "test-job-1",
		SessionID:    "session-1",
		Status:       preemptive.JobStatusQueued,
		CreatedAt:    now,
		MessageCount: 5,
		Model:        "claude-sonnet-4-5",
	}

	assert.Equal(t, "test-job-1", job.ID)
	assert.Equal(t, "session-1", job.SessionID)
	assert.Equal(t, preemptive.JobStatusQueued, job.Status)
	assert.Equal(t, 5, job.MessageCount)
	assert.Equal(t, "claude-sonnet-4-5", job.Model)
}

func TestSummarizationJob_WithMessages(t *testing.T) {
	// Test that messages can be assigned correctly
	messages := []json.RawMessage{
		json.RawMessage(`{"role": "user", "content": "Hello"}`),
		json.RawMessage(`{"role": "assistant", "content": "Hi there!"}`),
	}

	job := &preemptive.SummarizationJob{
		ID:           "test-job-2",
		SessionID:    "session-2",
		Status:       preemptive.JobStatusQueued,
		CreatedAt:    time.Now(),
		Messages:     messages,
		MessageCount: len(messages),
		Model:        "claude-sonnet-4-5",
	}

	assert.Equal(t, 2, job.MessageCount)
	assert.Len(t, job.Messages, 2)
}

func TestSummarizationJob_CompletedState(t *testing.T) {
	// Test job in completed state
	now := time.Now()
	startTime := now.Add(-5 * time.Second)
	completedTime := now

	job := &preemptive.SummarizationJob{
		ID:            "test-job-3",
		SessionID:     "session-3",
		Status:        preemptive.JobStatusCompleted,
		CreatedAt:     now.Add(-10 * time.Second),
		StartedAt:     &startTime,
		CompletedAt:   &completedTime,
		MessageCount:  10,
		Model:         "claude-sonnet-4-5",
		Summary:       "This is the summary of the conversation.",
		SummaryTokens: 150,
		LastIndex:     7,
	}

	assert.Equal(t, preemptive.JobStatusCompleted, job.Status)
	assert.NotNil(t, job.StartedAt)
	assert.NotNil(t, job.CompletedAt)
	assert.Equal(t, "This is the summary of the conversation.", job.Summary)
	assert.Equal(t, 150, job.SummaryTokens)
	assert.Equal(t, 7, job.LastIndex)
}

func TestSummarizationJob_FailedState(t *testing.T) {
	// Test job in failed state
	now := time.Now()
	startTime := now.Add(-5 * time.Second)
	completedTime := now

	job := &preemptive.SummarizationJob{
		ID:           "test-job-4",
		SessionID:    "session-4",
		Status:       preemptive.JobStatusFailed,
		CreatedAt:    now.Add(-10 * time.Second),
		StartedAt:    &startTime,
		CompletedAt:  &completedTime,
		MessageCount: 10,
		Model:        "claude-sonnet-4-5",
		Error:        "API rate limit exceeded",
	}

	assert.Equal(t, preemptive.JobStatusFailed, job.Status)
	assert.NotEmpty(t, job.Error)
}

func TestSummarizationJob_JSONSerialization(t *testing.T) {
	// Test that job can be serialized to JSON (for logging/monitoring)
	job := &preemptive.SummarizationJob{
		ID:           "test-job-5",
		SessionID:    "session-5",
		Status:       preemptive.JobStatusQueued,
		CreatedAt:    time.Now(),
		MessageCount: 3,
		Model:        "claude-sonnet-4-5",
	}

	data, err := json.Marshal(job)
	require.NoError(t, err)
	assert.Contains(t, string(data), "test-job-5")
	assert.Contains(t, string(data), "session-5")
	assert.Contains(t, string(data), "queued")
}
