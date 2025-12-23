// Package control implements the ControlService for unit lifecycle management.
package control

import (
	"sync"
	"sync/atomic"
	"time"

	latisv1 "github.com/shanemcd/latis/gen/go/latis/v1"
)

// State tracks the runtime state of a unit.
type State struct {
	state       atomic.Int32
	startTime   time.Time
	activeTasks atomic.Int32
	identity    string

	mu       sync.RWMutex
	metadata map[string]string
}

// NewState creates a new State in STARTING mode.
func NewState(identity string) *State {
	s := &State{
		startTime: time.Now(),
		identity:  identity,
		metadata:  make(map[string]string),
	}
	s.state.Store(int32(latisv1.UnitState_UNIT_STATE_STARTING))
	return s
}

// SetReady transitions to READY state.
func (s *State) SetReady() {
	s.state.Store(int32(latisv1.UnitState_UNIT_STATE_READY))
}

// SetDraining transitions to DRAINING state.
func (s *State) SetDraining() {
	s.state.Store(int32(latisv1.UnitState_UNIT_STATE_DRAINING))
}

// SetStopped transitions to STOPPED state.
func (s *State) SetStopped() {
	s.state.Store(int32(latisv1.UnitState_UNIT_STATE_STOPPED))
}

// IncrementTasks increments active task count and sets BUSY if currently READY.
func (s *State) IncrementTasks() {
	s.activeTasks.Add(1)
	s.state.CompareAndSwap(
		int32(latisv1.UnitState_UNIT_STATE_READY),
		int32(latisv1.UnitState_UNIT_STATE_BUSY),
	)
}

// DecrementTasks decrements active task count and sets READY if now zero and was BUSY.
func (s *State) DecrementTasks() {
	if s.activeTasks.Add(-1) == 0 {
		s.state.CompareAndSwap(
			int32(latisv1.UnitState_UNIT_STATE_BUSY),
			int32(latisv1.UnitState_UNIT_STATE_READY),
		)
	}
}

// GetState returns the current unit state enum.
func (s *State) GetState() latisv1.UnitState {
	return latisv1.UnitState(s.state.Load())
}

// GetActiveTasks returns the current active task count.
func (s *State) GetActiveTasks() int32 {
	return s.activeTasks.Load()
}

// GetUptime returns seconds since start.
func (s *State) GetUptime() int64 {
	return int64(time.Since(s.startTime).Seconds())
}

// GetIdentity returns the SPIFFE identity.
func (s *State) GetIdentity() string {
	return s.identity
}

// SetMetadata sets a metadata key-value pair.
func (s *State) SetMetadata(key, value string) {
	s.mu.Lock()
	s.metadata[key] = value
	s.mu.Unlock()
}

// GetMetadata returns a copy of the metadata map.
func (s *State) GetMetadata() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]string, len(s.metadata))
	for k, v := range s.metadata {
		result[k] = v
	}
	return result
}
