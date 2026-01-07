// Package ui provides the terminal user interface for beads_viewer.
// This file implements the BackgroundWorker for off-thread data processing.
package ui

import (
	"context"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Dicklesworthstone/beads_viewer/pkg/loader"
	"github.com/Dicklesworthstone/beads_viewer/pkg/watcher"
)

// WorkerState represents the current state of the background worker.
type WorkerState int

const (
	// WorkerIdle means the worker is waiting for file changes.
	WorkerIdle WorkerState = iota
	// WorkerProcessing means the worker is building a new snapshot.
	WorkerProcessing
	// WorkerStopped means the worker has been stopped.
	WorkerStopped
)

// BackgroundWorker manages background processing of beads data.
// It owns the file watcher, implements coalescing, and builds snapshots
// off the UI thread.
type BackgroundWorker struct {
	// Configuration
	beadsPath     string
	debounceDelay time.Duration

	// State
	mu       sync.RWMutex
	state    WorkerState
	dirty    bool // True if a change came in while processing
	snapshot *DataSnapshot
	started  bool // True if Start() has been called

	// Components
	watcher *watcher.Watcher
	program *tea.Program

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

// WorkerConfig configures the BackgroundWorker.
type WorkerConfig struct {
	BeadsPath     string
	DebounceDelay time.Duration
	Program       *tea.Program
}

// NewBackgroundWorker creates a new background worker.
func NewBackgroundWorker(cfg WorkerConfig) (*BackgroundWorker, error) {
	ctx, cancel := context.WithCancel(context.Background())

	if cfg.DebounceDelay == 0 {
		cfg.DebounceDelay = 200 * time.Millisecond
	}

	w := &BackgroundWorker{
		beadsPath:     cfg.BeadsPath,
		debounceDelay: cfg.DebounceDelay,
		program:       cfg.Program,
		state:         WorkerIdle,
		ctx:           ctx,
		cancel:        cancel,
		done:          make(chan struct{}),
	}

	// Initialize file watcher
	if cfg.BeadsPath != "" {
		fw, err := watcher.NewWatcher(cfg.BeadsPath,
			watcher.WithDebounceDuration(cfg.DebounceDelay),
		)
		if err != nil {
			cancel()
			return nil, err
		}
		w.watcher = fw
	}

	return w, nil
}

// Start begins watching for file changes and processing in the background.
// Start is idempotent - calling it multiple times has no effect.
func (w *BackgroundWorker) Start() error {
	w.mu.Lock()
	if w.started {
		w.mu.Unlock()
		return nil // Already started
	}
	w.started = true
	w.mu.Unlock()

	if w.watcher != nil {
		if err := w.watcher.Start(); err != nil {
			return err
		}

		// Start the processing loop
		go w.processLoop()
	} else {
		// No watcher - close done channel immediately so Stop() doesn't block
		close(w.done)
	}

	return nil
}

// Stop halts the background worker and cleans up resources.
// Stop is idempotent - calling it multiple times has no effect.
func (w *BackgroundWorker) Stop() {
	w.mu.Lock()
	if w.state == WorkerStopped {
		w.mu.Unlock()
		return
	}
	w.state = WorkerStopped
	wasStarted := w.started
	w.mu.Unlock()

	w.cancel()

	if w.watcher != nil {
		w.watcher.Stop()
	}

	// Only wait for done if Start() was called
	if wasStarted {
		select {
		case <-w.done:
		case <-time.After(2 * time.Second):
			// Timeout waiting for graceful shutdown
		}
	}
}

// TriggerRefresh manually triggers a refresh of the data.
// Has no effect if the worker is stopped or already processing.
func (w *BackgroundWorker) TriggerRefresh() {
	w.mu.Lock()
	if w.state == WorkerStopped {
		w.mu.Unlock()
		return
	}
	if w.state == WorkerProcessing {
		w.dirty = true
		w.mu.Unlock()
		return
	}
	w.mu.Unlock()

	// Trigger processing
	go w.process()
}

// GetSnapshot returns the current snapshot (may be nil).
func (w *BackgroundWorker) GetSnapshot() *DataSnapshot {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.snapshot
}

// State returns the current worker state.
func (w *BackgroundWorker) State() WorkerState {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.state
}

// processLoop watches for file changes and triggers processing.
func (w *BackgroundWorker) processLoop() {
	defer close(w.done)

	if w.watcher == nil {
		return
	}

	for {
		select {
		case <-w.ctx.Done():
			return

		case <-w.watcher.Changed():
			w.process()
		}
	}
}

// process builds a new snapshot from the current file.
func (w *BackgroundWorker) process() {
	w.mu.Lock()
	if w.state == WorkerStopped {
		w.mu.Unlock()
		return
	}
	w.state = WorkerProcessing
	w.dirty = false
	w.mu.Unlock()

	// Load and build snapshot
	snapshot := w.buildSnapshot()

	w.mu.Lock()
	// Check if stopped while we were processing - don't overwrite stopped state
	if w.state == WorkerStopped {
		w.mu.Unlock()
		return
	}
	w.snapshot = snapshot
	wasDirty := w.dirty
	w.state = WorkerIdle
	w.mu.Unlock()

	// Notify UI
	if w.program != nil && snapshot != nil {
		w.program.Send(SnapshotReadyMsg{Snapshot: snapshot})
	}

	// If dirty, process again immediately
	if wasDirty {
		go w.process()
	}
}

// buildSnapshot loads data and constructs a new DataSnapshot.
func (w *BackgroundWorker) buildSnapshot() *DataSnapshot {
	if w.beadsPath == "" {
		return nil
	}

	// Load issues from file
	issues, err := loader.LoadIssuesFromFile(w.beadsPath)
	if err != nil {
		// TODO: Send error message to UI
		return nil
	}

	// Build snapshot
	builder := NewSnapshotBuilder(issues)
	return builder.Build()
}

// SnapshotReadyMsg is sent to the UI when a new snapshot is ready.
type SnapshotReadyMsg struct {
	Snapshot *DataSnapshot
}

// Phase2UpdateMsg is sent when Phase 2 analysis completes.
// This allows the UI to update without waiting for full rebuild.
type Phase2UpdateMsg struct {
	// Phase 2 metrics are embedded in the GraphStats
}

// WatcherChanged returns the watcher's change notification channel.
// This is useful for integration with existing code.
func (w *BackgroundWorker) WatcherChanged() <-chan struct{} {
	if w.watcher == nil {
		return nil
	}
	return w.watcher.Changed()
}
