package logger

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/fsnotify/fsnotify"
	"k8s.io/klog/v2"

	v1 "k8s.io/api/core/v1"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
	"k8s.io/kubernetes/pkg/kubelet/types"
	"k8s.io/kubernetes/pkg/util/tail"
)

const (
	// timeFormatIn is the format for parsing timestamps from other logs.
	timeFormatIn = types.RFC3339NanoLenient

	// logForceCheckPeriod is the period to check for a new read
	logForceCheckPeriod = 1 * time.Second
)

var (
	// eol is the end-of-line sign in the log.
	eol = []byte{'\n'}
	// delimiter is the delimiter for timestamp and stream type in log line.
	delimiter = []byte{' '}
	// tagDelimiter is the delimiter for log tags.
	tagDelimiter = []byte(runtimeapi.LogTagDelimiter)
)

// LogOptions is the CRI internal type of all log options.
type LogOptions struct {
	tail      int64
	bytes     int64
	since     time.Time
	follow    bool
	timestamp bool
}

// NewLogOptions convert the v1.PodLogOptions to CRI internal LogOptions.
func NewLogOptions(apiOpts *v1.PodLogOptions, now time.Time) *LogOptions {
	opts := &LogOptions{
		tail:      -1, // -1 by default which means read all logs.
		bytes:     -1, // -1 by default which means read all logs.
		follow:    apiOpts.Follow,
		timestamp: apiOpts.Timestamps,
	}
	if apiOpts.TailLines != nil {
		opts.tail = *apiOpts.TailLines
	}
	if apiOpts.LimitBytes != nil {
		opts.bytes = *apiOpts.LimitBytes
	}
	if apiOpts.SinceSeconds != nil {
		opts.since = now.Add(-time.Duration(*apiOpts.SinceSeconds) * time.Second)
	}
	if apiOpts.SinceTime != nil && apiOpts.SinceTime.After(opts.since) {
		opts.since = apiOpts.SinceTime.Time
	}
	return opts
}

// parseFunc is a function parsing one log line to the internal log type.
// Notice that the caller must make sure logMessage is not nil.
type parseFunc func([]byte, *Message) error

var parseFuncs = []parseFunc{
	parseCRILog, // CRI log format parse function
}

// parseCRILog parses logs in CRI log format. CRI Log format example:
//
//	2016-10-06T00:17:09.669794202Z stdout P log content 1
//	2016-10-06T00:17:09.669794203Z stderr F log content 2
func parseCRILog(log []byte, msg *Message) error {
	var err error
	// Parse timestamp
	idx := bytes.Index(log, delimiter)
	if idx < 0 {
		return fmt.Errorf("timestamp is not found")
	}
	msg.Timestamp, err = time.Parse(timeFormatIn, string(log[:idx]))
	if err != nil {
		return fmt.Errorf("unexpected timestamp format %q: %v", timeFormatIn, err)
	}

	// Parse stream type
	log = log[idx+1:]
	idx = bytes.Index(log, delimiter)
	if idx < 0 {
		return fmt.Errorf("stream type is not found")
	}

	// Parse log tag
	log = log[idx+1:]
	idx = bytes.Index(log, delimiter)
	if idx < 0 {
		return fmt.Errorf("log tag is not found")
	}
	// Keep this forward compatible.
	tags := bytes.Split(log[:idx], tagDelimiter)
	partial := runtimeapi.LogTag(tags[0]) == runtimeapi.LogTagPartial
	// Trim the tailing new line if this is a partial line.
	if partial && len(log) > 0 && log[len(log)-1] == '\n' {
		log = log[:len(log)-1]
	}

	// Get log content
	msg.Line = log[idx+1:]

	return nil
}

// getParseFunc returns proper parse function based on the sample log line passed in.
func getParseFunc(log []byte) (parseFunc, error) {
	for _, p := range parseFuncs {
		if err := p(log, &Message{}); err == nil {
			return p, nil
		}
	}
	return nil, fmt.Errorf("unsupported log format: %q", log)
}

// ReadLogs read the container log and redirect into stdout and stderr.
// Note that containerID is only needed when following the log, or else
// just pass in empty string "".
func ReadLogs(ctx context.Context, path, containerID string, opts *ReadConfig, runtimeService runtimeapi.RuntimeServiceClient, watch *LogWatcher) error {
	// fsnotify has different behavior for symlinks in different platform,
	// for example it follows symlink on Linux, but not on Windows,
	// so we explicitly resolve symlinks before reading the logs.
	// There shouldn't be security issue because the container log
	// path is owned by kubelet and the container runtime.
	evaluated, err := filepath.EvalSymlinks(path)
	if err != nil {
		return fmt.Errorf("failed to try resolving symlinks in path %q: %v", path, err)
	}
	path = evaluated
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open log file %q: %v", path, err)
	}
	defer f.Close()

	// Search start point based on tail line.
	start, err := tail.FindTailLineStartIndex(f, int64(opts.Tail))
	if err != nil {
		return fmt.Errorf("failed to tail %d lines of log file %q: %v", opts.Tail, path, err)
	}
	if _, err := f.Seek(start, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek %d in log file %q: %v", start, path, err)
	}

	limitedMode := (opts.Tail >= 0) && (!opts.Follow)
	limitedNum := opts.Tail
	// Start parsing the logs.
	r := bufio.NewReader(f)
	// Do not create watcher here because it is not needed if `Follow` is false.
	var watcher *fsnotify.Watcher
	var parse parseFunc
	var stop bool
	found := true
	msg := &Message{}
	for {
		if stop || (limitedMode && limitedNum == 0) {
			klog.V(2).InfoS("Finished parsing log file", "path", path)
			return nil
		}
		l, err := r.ReadBytes(eol[0])
		if err != nil {
			if err != io.EOF { // This is an real error
				return fmt.Errorf("failed to read log file %q: %v", path, err)
			}
			if opts.Follow {
				// The container is not running, we got to the end of the log.
				if !found {
					return nil
				}
				// Reset seek so that if this is an incomplete line,
				// it will be read again.
				if _, err := f.Seek(-int64(len(l)), io.SeekCurrent); err != nil {
					return fmt.Errorf("failed to reset seek in log file %q: %v", path, err)
				}
				if watcher == nil {
					// Initialize the watcher if it has not been initialized yet.
					if watcher, err = fsnotify.NewWatcher(); err != nil {
						return fmt.Errorf("failed to create fsnotify watcher: %v", err)
					}
					defer watcher.Close()
					if err := watcher.Add(f.Name()); err != nil {
						return fmt.Errorf("failed to watch file %q: %v", f.Name(), err)
					}
					// If we just created the watcher, try again to read as we might have missed
					// the event.
					continue
				}
				var recreated bool
				// Wait until the next log change.
				found, recreated, err = waitLogs(ctx, containerID, watcher, runtimeService)
				if err != nil {
					return err
				}
				if recreated {
					newF, err := os.Open(path)
					if err != nil {
						if os.IsNotExist(err) {
							continue
						}
						return fmt.Errorf("failed to open log file %q: %v", path, err)
					}
					defer newF.Close()
					f.Close()
					if err := watcher.Remove(f.Name()); err != nil && !os.IsNotExist(err) {
						klog.ErrorS(err, "Failed to remove file watch", "path", f.Name())
					}
					f = newF
					if err := watcher.Add(f.Name()); err != nil {
						return fmt.Errorf("failed to watch file %q: %v", f.Name(), err)
					}
					r = bufio.NewReader(f)
				}
				// If the container exited consume data until the next EOF
				continue
			}
			// Should stop after writing the remaining content.
			stop = true
			if len(l) == 0 {
				continue
			}
			logrus.Warn("Incomplete line in log file", "path", path, "line", l)
		}
		if parse == nil {
			// Initialize the log parsing function.
			parse, err = getParseFunc(l)
			if err != nil {
				return fmt.Errorf("failed to get parse function: %v", err)
			}
		}
		// Parse the log line.
		msg.reset()
		if err := parse(l, msg); err != nil {
			klog.ErrorS(err, "Failed when parsing line in log file", "path", path, "line", l)
			continue
		}
		msgTemporary := *msg
		watch.Msg <- &msgTemporary
		watch.Msg <- msg
		if limitedMode {
			limitedNum--
		}
	}
}

func isContainerRunning(id string, r runtimeapi.RuntimeServiceClient) (bool, error) {
	resp, err := r.ContainerStatus(context.Background(), &runtimeapi.ContainerStatusRequest{
		ContainerId: id,
	})
	if err != nil {
		return false, err
	}
	// Only keep following container log when it is running.
	if resp.GetStatus().GetState().String() != runtimeapi.ContainerState_CONTAINER_RUNNING.String() {
		logrus.Infoln("Container is not running", "containerId", id, "state", resp.GetStatus().GetState().String())
		// Do not return error because it's normal that the container stops
		// during waiting.
		return false, nil
	}
	return true, nil
}

// waitLogs wait for the next log write. It returns two booleans and an error. The first boolean
// indicates whether a new log is found; the second boolean if the log file was recreated;
// the error is error happens during waiting new logs.
func waitLogs(ctx context.Context, id string, w *fsnotify.Watcher, runtimeService runtimeapi.RuntimeServiceClient) (bool, bool, error) {
	// no need to wait if the pod is not running
	if running, err := isContainerRunning(id, runtimeService); !running {
		return false, false, err
	}
	errRetry := 5
	tC := time.After(logForceCheckPeriod)
	for {
		select {
		case <-ctx.Done():
			return false, false, fmt.Errorf("context cancelled")
		case e := <-w.Events:
			switch e.Op {
			case fsnotify.Write:
				return true, false, nil
			case fsnotify.Create:
				fallthrough
			case fsnotify.Rename:
				fallthrough
			case fsnotify.Remove:
				fallthrough
			case fsnotify.Chmod:
				return true, true, nil
			default:
				klog.ErrorS(nil, "Received unexpected fsnotify event, retrying", "event", e)
			}
		case err := <-w.Errors:
			klog.ErrorS(err, "Received fsnotify watch error, retrying unless no more retries left", "retries", errRetry)
			if errRetry == 0 {
				return false, false, err
			}
			errRetry--
		case <-tC:
			return true, false, nil
		}
	}
}
