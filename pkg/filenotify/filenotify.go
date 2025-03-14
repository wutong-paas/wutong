package filenotify

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
)

// FileWatcher is an interface for implementing file notification watchers
type FileWatcher interface {
	Events() <-chan fsnotify.Event
	Errors() <-chan error
	Add(name string) error
	Remove(name string) error
	Close() error
}

// New tries to use an fs-event watcher, and falls back to the poller if there is an error
func New() (FileWatcher, error) {
	if watcher, err := NewEventWatcher(); err == nil {
		return watcher, nil
	}
	return NewPollingWatcher(), nil
}

// NewPollingWatcher returns a poll-based file watcher
func NewPollingWatcher() FileWatcher {
	return &filePoller{
		events: make(chan fsnotify.Event),
		errors: make(chan error),
	}
}

// NewEventWatcher returns an fs-event based file watcher
func NewEventWatcher() (FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &fsNotifyWatcher{watcher}, nil
}

// fsNotifyWatcher wraps the fsnotify package to satisfy the FileNotifier interface
type fsNotifyWatcher struct {
	*fsnotify.Watcher
}

// Events returns the fsnotify event channel receiver
func (w *fsNotifyWatcher) Events() <-chan fsnotify.Event {
	return w.Watcher.Events
}

// Errors returns the fsnotify error channel receiver
func (w *fsNotifyWatcher) Errors() <-chan error {
	return w.Watcher.Errors
}

var (
	// errPollerClosed is returned when the poller is closed
	errPollerClosed = errors.New("poller is closed")
	// errNoSuchWatch is returned when trying to remove a watch that doesn't exist
	errNoSuchWatch = errors.New("watch does not exist")
)

// watchWaitTime is the time to wait between file poll loops
const watchWaitTime = 200 * time.Millisecond

// filePoller is used to poll files for changes, especially in cases where fsnotify
// can't be run (e.g. when inotify handles are exhausted)
// filePoller satisfies the FileWatcher interface
type filePoller struct {
	// watches is the list of files currently being polled, close the associated channel to stop the watch
	watches map[string]chan struct{}
	// events is the channel to listen to for watch events
	events chan fsnotify.Event
	// errors is the channel to listen to for watch errors
	errors chan error
	// mu locks the poller for modification
	mu sync.Mutex
	// closed is used to specify when the poller has already closed
	closed bool
}

// Add adds a filename to the list of watches
// once added the file is polled for changes in a separate goroutine
func (w *filePoller) Add(name string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return errPollerClosed
	}
	f, err := os.Open(name)
	if err != nil {
		return err
	}
	fi, err := os.Stat(name)
	if err != nil {
		f.Close()
		return err
	}
	if w.watches == nil {
		w.watches = make(map[string]chan struct{})
	}
	if _, exists := w.watches[name]; exists {
		f.Close()
		return fmt.Errorf("watch exists")
	}
	chClose := make(chan struct{})
	w.watches[name] = chClose
	go w.watch(f, fi, chClose)
	return nil
}

// Remove stops and removes watch with the specified name
func (w *filePoller) Remove(name string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.remove(name)
}
func (w *filePoller) remove(name string) error {
	if w.closed {
		return errPollerClosed
	}
	chClose, exists := w.watches[name]
	if !exists {
		return errNoSuchWatch
	}
	close(chClose)
	delete(w.watches, name)
	return nil
}

// Events returns the event channel
// This is used for notifications on events about watched files
func (w *filePoller) Events() <-chan fsnotify.Event {
	return w.events
}

// Errors returns the errors channel
// This is used for notifications about errors on watched files
func (w *filePoller) Errors() <-chan error {
	return w.errors
}

// Close closes the poller
// All watches are stopped, removed, and the poller cannot be added to
func (w *filePoller) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return nil
	}
	for name := range w.watches {
		w.remove(name)
	}
	w.closed = true
	return nil
}

// sendEvent publishes the specified event to the events channel
func (w *filePoller) sendEvent(e fsnotify.Event, chClose <-chan struct{}) error {
	select {
	case w.events <- e:
	case <-chClose:
		return fmt.Errorf("closed")
	}
	return nil
}

// sendErr publishes the specified error to the errors channel
func (w *filePoller) sendErr(e error, chClose <-chan struct{}) error {
	select {
	case w.errors <- e:
	case <-chClose:
		return fmt.Errorf("closed")
	}
	return nil
}

// watch is responsible for polling the specified file for changes
// upon finding changes to a file or errors, sendEvent/sendErr is called
func (w *filePoller) watch(f *os.File, lastFi os.FileInfo, chClose chan struct{}) {
	defer f.Close()
	timer := time.NewTimer(watchWaitTime)
	if !timer.Stop() {
		<-timer.C
	}
	defer timer.Stop()
	for {
		timer.Reset(watchWaitTime)
		select {
		case <-timer.C:
		case <-chClose:
			logrus.Debugf("watch for %s closed", f.Name())
			return
		}
		fi, err := os.Stat(f.Name())
		if err != nil {
			// if we got an error here and lastFi is not set, we can presume that nothing has changed
			// This should be safe since before `watch()` is called, a stat is performed, there is any error `watch` is not called
			if lastFi == nil {
				continue
			}
			// If it doesn't exist at this point, it must have been removed
			// no need to send the error here since this is a valid operation
			if os.IsNotExist(err) {
				if err := w.sendEvent(fsnotify.Event{Op: fsnotify.Remove, Name: f.Name()}, chClose); err != nil {
					return
				}
				lastFi = nil
				continue
			}
			// at this point, send the error
			if err := w.sendErr(err, chClose); err != nil {
				return
			}
			continue
		}
		if lastFi == nil {
			if err := w.sendEvent(fsnotify.Event{Op: fsnotify.Create, Name: fi.Name()}, chClose); err != nil {
				return
			}
			lastFi = fi
			continue
		}
		if fi.Mode() != lastFi.Mode() {
			if err := w.sendEvent(fsnotify.Event{Op: fsnotify.Chmod, Name: fi.Name()}, chClose); err != nil {
				return
			}
			lastFi = fi
			continue
		}
		if fi.ModTime() != lastFi.ModTime() || fi.Size() != lastFi.Size() {
			if err := w.sendEvent(fsnotify.Event{Op: fsnotify.Write, Name: fi.Name()}, chClose); err != nil {
				return
			}
			lastFi = fi
			continue
		}
	}
}
