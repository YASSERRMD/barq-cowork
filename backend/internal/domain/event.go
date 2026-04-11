package domain

import "time"

// EventType classifies a domain event emitted during task execution.
type EventType string

const (
	EventTypeTaskCreated    EventType = "task.created"
	EventTypeTaskStarted    EventType = "task.started"
	EventTypeTaskCompleted  EventType = "task.completed"
	EventTypeTaskFailed     EventType = "task.failed"
	EventTypeStepStarted    EventType = "step.started"
	EventTypeStepCompleted  EventType = "step.completed"
	EventTypeToolCalled     EventType = "tool.called"
	EventTypeToolResult     EventType = "tool.result"
	EventTypeApprovalNeeded EventType = "approval.needed"
	EventTypeArtifactReady  EventType = "artifact.ready"
	EventTypeLogLine        EventType = "log.line"
)

// Event is an immutable audit record of something that happened during
// task execution. It feeds the live timeline UI and the diagnostic bundle.
type Event struct {
	ID        string
	TaskID    string
	Type      EventType
	Payload   string // JSON-encoded event-specific data
	CreatedAt time.Time
}
