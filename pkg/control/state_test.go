package control

import (
	"sync"
	"testing"

	latisv1 "github.com/shanemcd/latis/gen/go/latis/v1"
)

func TestNewState(t *testing.T) {
	s := NewState("spiffe://latis/unit/test")

	if s.GetState() != latisv1.UnitState_UNIT_STATE_STARTING {
		t.Errorf("expected STARTING state, got %v", s.GetState())
	}
	if s.GetIdentity() != "spiffe://latis/unit/test" {
		t.Errorf("expected identity 'spiffe://latis/unit/test', got %v", s.GetIdentity())
	}
	if s.GetActiveTasks() != 0 {
		t.Errorf("expected 0 active tasks, got %v", s.GetActiveTasks())
	}
}

func TestStateTransitions(t *testing.T) {
	s := NewState("test")

	// STARTING -> READY
	s.SetReady()
	if s.GetState() != latisv1.UnitState_UNIT_STATE_READY {
		t.Errorf("expected READY state, got %v", s.GetState())
	}

	// READY -> DRAINING
	s.SetDraining()
	if s.GetState() != latisv1.UnitState_UNIT_STATE_DRAINING {
		t.Errorf("expected DRAINING state, got %v", s.GetState())
	}

	// DRAINING -> STOPPED
	s.SetStopped()
	if s.GetState() != latisv1.UnitState_UNIT_STATE_STOPPED {
		t.Errorf("expected STOPPED state, got %v", s.GetState())
	}
}

func TestTaskTracking(t *testing.T) {
	s := NewState("test")
	s.SetReady()

	// Increment tasks: READY -> BUSY
	s.IncrementTasks()
	if s.GetActiveTasks() != 1 {
		t.Errorf("expected 1 active task, got %v", s.GetActiveTasks())
	}
	if s.GetState() != latisv1.UnitState_UNIT_STATE_BUSY {
		t.Errorf("expected BUSY state, got %v", s.GetState())
	}

	// Add another task
	s.IncrementTasks()
	if s.GetActiveTasks() != 2 {
		t.Errorf("expected 2 active tasks, got %v", s.GetActiveTasks())
	}

	// Decrement (still busy)
	s.DecrementTasks()
	if s.GetActiveTasks() != 1 {
		t.Errorf("expected 1 active task, got %v", s.GetActiveTasks())
	}
	if s.GetState() != latisv1.UnitState_UNIT_STATE_BUSY {
		t.Errorf("expected BUSY state, got %v", s.GetState())
	}

	// Decrement to zero: BUSY -> READY
	s.DecrementTasks()
	if s.GetActiveTasks() != 0 {
		t.Errorf("expected 0 active tasks, got %v", s.GetActiveTasks())
	}
	if s.GetState() != latisv1.UnitState_UNIT_STATE_READY {
		t.Errorf("expected READY state, got %v", s.GetState())
	}
}

func TestTaskTrackingConcurrent(t *testing.T) {
	s := NewState("test")
	s.SetReady()

	var wg sync.WaitGroup
	n := 100

	// Increment concurrently
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.IncrementTasks()
		}()
	}
	wg.Wait()

	if s.GetActiveTasks() != int32(n) {
		t.Errorf("expected %d active tasks, got %v", n, s.GetActiveTasks())
	}

	// Decrement concurrently
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.DecrementTasks()
		}()
	}
	wg.Wait()

	if s.GetActiveTasks() != 0 {
		t.Errorf("expected 0 active tasks, got %v", s.GetActiveTasks())
	}
	if s.GetState() != latisv1.UnitState_UNIT_STATE_READY {
		t.Errorf("expected READY state after all tasks complete, got %v", s.GetState())
	}
}

func TestMetadata(t *testing.T) {
	s := NewState("test")

	s.SetMetadata("version", "1.0.0")
	s.SetMetadata("build", "abc123")

	meta := s.GetMetadata()
	if meta["version"] != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %v", meta["version"])
	}
	if meta["build"] != "abc123" {
		t.Errorf("expected build 'abc123', got %v", meta["build"])
	}

	// Verify we get a copy, not the original
	meta["version"] = "modified"
	meta2 := s.GetMetadata()
	if meta2["version"] != "1.0.0" {
		t.Errorf("metadata should be a copy, got modified value")
	}
}

func TestUptime(t *testing.T) {
	s := NewState("test")

	uptime := s.GetUptime()
	if uptime < 0 {
		t.Errorf("uptime should be non-negative, got %v", uptime)
	}
}
