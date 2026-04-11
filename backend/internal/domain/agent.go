package domain

import "time"

// AgentRole classifies the specialisation of a sub-agent.
type AgentRole string

const (
	AgentRoleResearcher AgentRole = "researcher"
	AgentRoleWriter     AgentRole = "writer"
	AgentRoleCoder      AgentRole = "coder"
	AgentRoleReviewer   AgentRole = "reviewer"
	AgentRoleAnalyst    AgentRole = "analyst"
	AgentRoleCustom     AgentRole = "custom"
)

// SubAgent is a specialised child agent spawned from a parent task.
// It runs its own Planner+Executor pipeline focused on a single role.
type SubAgent struct {
	ID           string
	ParentTaskID string    // the task that spawned this agent
	Role         AgentRole // determines the system-prompt flavour
	Title        string    // human-readable description of this agent's job
	Instructions string    // focused sub-task text passed to the planner
	Status       TaskStatus
	PlanID       string // populated after planning; empty until then
	CreatedAt    time.Time
	UpdatedAt    time.Time
	StartedAt    *time.Time
	CompletedAt  *time.Time
}

// SubAgentSpec is the input used to create a sub-agent.
type SubAgentSpec struct {
	Role         AgentRole
	Title        string
	Instructions string
}

// RoleSystemPrompt returns a focused system-prompt prefix for each role.
// The full planner system prompt is appended after this prefix.
func (r AgentRole) SystemPromptPrefix() string {
	switch r {
	case AgentRoleResearcher:
		return "You are a research sub-agent. Your sole focus is gathering information, " +
			"reading files, and producing a structured summary. "
	case AgentRoleWriter:
		return "You are a writing sub-agent. Your sole focus is producing well-structured " +
			"markdown documents, reports, and summaries. "
	case AgentRoleCoder:
		return "You are a coding sub-agent. Your sole focus is reading existing code, " +
			"writing new code, and producing working file outputs. "
	case AgentRoleReviewer:
		return "You are a review sub-agent. Your sole focus is reading existing work, " +
			"identifying issues, and producing a structured review report. "
	case AgentRoleAnalyst:
		return "You are an analysis sub-agent. Your sole focus is processing data, " +
			"identifying patterns, and producing structured JSON or markdown analysis outputs. "
	default:
		return ""
	}
}
