package task

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	"nirenjan.org/overlord/config"
)

type State int

const (
	InProgress State = iota
	Assigned
	Blocked
	Deferred
	Completed
	Deleted
)

func (s State) String() string {
	switch s {
	case Assigned:
		return "assigned"
	case InProgress:
		return "in-progress"
	case Blocked:
		return "blocked"
	case Deferred:
		return "deferred"
	case Completed:
		return "complete"
	case Deleted:
		return "obsolete"
	}

	return fmt.Sprintf("State(%d)", s)
}

type Task struct {
	ID          string        `json:"-"`
	Created     time.Time     `json:"created"`
	State       State         `json:"state"`
	Due         time.Time     `json:"due"`
	Priority    int           `json:"priority"`
	Description string        `json:"description"`
	Notes       string        `json:"notes,omitempty"`
	Started     time.Time     `json:"started,omitempty"`
	Worked      time.Duration `json:"worked,omitempty"`
	Path        string        `json:"-"`
}

func parsePriority(prio string) (priority int, err error) {
	priority, err = strconv.Atoi(prio)
	if err != nil || priority < 0 || priority > 9 {
		err = fmt.Errorf("Priority must be between 0 and 9, got '%v'", prio)
	}

	return
}

// Allowed state transitions
// Assigned -> InProgress, Blocked, Deferred, Deleted
// InProgress -> Assigned, Blocked, Completed, Deferred
// Blocked -> InProgress, Deferred, Deleted
// Deferred -> InProgress, Blocked, Deleted
// Completed -> _
// Deleted -> _
func (t *Task) stateTransition(newState State) error {
	var allowed bool
	switch t.State {
	case Assigned:
		switch newState {
		case InProgress, Blocked, Deferred, Deleted:
			allowed = true
		}

	case InProgress:
		switch newState {
		case Assigned, Blocked, Completed, Deferred:
			allowed = true
		}

	case Blocked:
		switch newState {
		case InProgress, Deferred, Deleted:
			allowed = true
		}

	case Deferred:
		switch newState {
		case InProgress, Blocked, Deleted:
			allowed = true
		}
	}

	if !allowed {
		return fmt.Errorf("Cannot transition task from %v to %v", t.State, newState)
	}

	if newState == InProgress {
		t.Started = time.Now()
	} else if t.State == InProgress {
		t.Worked += time.Since(t.Started)
		t.Started = time.Time{}
	}

	t.State = newState
	return nil
}

// UpdateID updates the ID for the task. Right now, this is solely based
// on the creation time, so that it doesn't change when the user changes
// the due date or any other field.
func (t *Task) UpdateID() {
	data := []byte(t.Created.Format(time.RFC3339))
	t.ID = fmt.Sprintf("%x", sha256.Sum256(data))[:10]
}

// UpdatePath updates the path for the backing file.
func (t *Task) UpdatePath() error {
	modDir, err := config.ModuleDir("task", t.Created.Format("2006"))
	if err != nil {
		return err
	}

	t.Path = filepath.Join(modDir, t.Created.Format("0102-150405.task"))
	return nil
}
