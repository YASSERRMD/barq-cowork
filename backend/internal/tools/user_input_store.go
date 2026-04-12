package tools

import (
	"context"
	"sync"
	"time"
)

// PendingInput holds an open channel waiting for the user's typed answer.
type PendingInput struct {
	TaskID   string
	Question string
	ch       chan string
}

// UserInputStore is an in-process registry that lets ask_user block the agent
// loop until the frontend POSTs a response.  It is safe for concurrent use.
type UserInputStore struct {
	mu    sync.Mutex
	items map[string]*PendingInput // keyed by pendingID (question UUID)
}

// NewUserInputStore creates an empty store.
func NewUserInputStore() *UserInputStore {
	return &UserInputStore{items: make(map[string]*PendingInput)}
}

// Register adds a pending question and returns its channel.
// The caller blocks on <-ch until Answer is called or ctx is cancelled.
func (s *UserInputStore) Register(pendingID, taskID, question string) chan string {
	ch := make(chan string, 1)
	s.mu.Lock()
	s.items[pendingID] = &PendingInput{TaskID: taskID, Question: question, ch: ch}
	s.mu.Unlock()
	return ch
}

// Answer delivers a user response to the waiting ask_user call.
// Returns true if the pendingID was found (and not already answered).
func (s *UserInputStore) Answer(pendingID, answer string) bool {
	s.mu.Lock()
	p, ok := s.items[pendingID]
	if ok {
		delete(s.items, pendingID)
	}
	s.mu.Unlock()
	if !ok {
		return false
	}
	select {
	case p.ch <- answer:
	default:
	}
	return true
}

// Wait blocks until the user answers or the context/timeout fires.
// Returns the answer and true, or ("", false) on timeout/cancel.
func (s *UserInputStore) Wait(ctx context.Context, ch chan string, timeout time.Duration) (string, bool) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case ans := <-ch:
		return ans, true
	case <-timer.C:
		return "", false
	case <-ctx.Done():
		return "", false
	}
}

// List returns all pending question IDs for a task (for the API to enumerate).
func (s *UserInputStore) List(taskID string) []PendingQuestion {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []PendingQuestion
	for id, p := range s.items {
		if p.TaskID == taskID {
			out = append(out, PendingQuestion{ID: id, Question: p.Question})
		}
	}
	return out
}

// PendingQuestion is the API-visible representation.
type PendingQuestion struct {
	ID       string `json:"id"`
	Question string `json:"question"`
}
