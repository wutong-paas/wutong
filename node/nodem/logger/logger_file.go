// WUTONG, Application Management Platform
// Copyright (C) 2014-2019 Wutong Co., Ltd.

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version. For any non-GPL usage of Wutong,
// one or multiple Commercial Licenses authorized by Wutong Co., Ltd.
// must be obtained first.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package logger

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/daemon/logger/jsonfilelog/jsonlog"
	"github.com/docker/docker/pkg/pools"
	"github.com/docker/docker/pkg/tailfile"
	"github.com/fsnotify/fsnotify"
	"github.com/moby/pubsub"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/pkg/filenotify"
)

const maxJSONDecodeRetry = 20000
const tmpLogfileSuffix = ".tmp"

// rotateFileMetadata is a metadata of the gzip header of the compressed log file
type rotateFileMetadata struct {
	LastTime time.Time `json:"lastTime,omitempty"`
}

// refCounter is a counter of logfile being referenced
type refCounter struct {
	mu      sync.Mutex
	counter map[string]int
}

// Reference increase the reference counter for specified logfile
func (rc *refCounter) GetReference(fileName string, openRefFile func(fileName string, exists bool) (*os.File, error)) (*os.File, error) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	var (
		file *os.File
		err  error
	)
	_, ok := rc.counter[fileName]
	file, err = openRefFile(fileName, ok)
	if err != nil {
		return nil, err
	}

	if ok {
		rc.counter[fileName]++
	} else if file != nil {
		rc.counter[file.Name()] = 1
	}

	return file, nil
}

// Dereference reduce the reference counter for specified logfile
func (rc *refCounter) Dereference(fileName string) error {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.counter[fileName]--
	if rc.counter[fileName] <= 0 {
		delete(rc.counter, fileName)
		err := os.Remove(fileName)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetTailReaderFunc is used to truncate a reader to only read as much as is required
// in order to get the passed in number of log lines.
// It returns the sectioned reader, the number of lines that the section reader
// contains, and any error that occurs.
type GetTailReaderFunc func(ctx context.Context, f SizeReaderAt, nLogLines int) (rdr io.Reader, nLines int, err error)

type makeDecoderFunc func(rdr io.Reader) func() (*Message, error)

type wrappedReaderAt struct {
	io.ReaderAt
	pos int64
}

func (r *wrappedReaderAt) Read(p []byte) (int, error) {
	n, err := r.ReaderAt.ReadAt(p, r.pos)
	r.pos += int64(n)
	return n, err
}

func decodeLogLine(dec *json.Decoder, l *jsonlog.JSONLog) (*Message, error) {
	l.Reset()
	if err := dec.Decode(l); err != nil {
		return nil, err
	}
	msg := &Message{
		Source:    l.Stream,
		Timestamp: l.Created,
		Line:      []byte(l.Log),
		Attrs:     l.Attrs,
	}
	return msg, nil
}

// decodeFunc is used to create a decoder for the log file reader
func decodeFunc(rdr io.Reader) func() (*Message, error) {
	l := &jsonlog.JSONLog{}
	dec := json.NewDecoder(rdr)
	return func() (msg *Message, err error) {
		for retries := 0; retries < maxJSONDecodeRetry; retries++ {
			msg, err = decodeLogLine(dec, l)
			if err == nil || err == io.EOF {
				break
			}
			// try again, could be due to a an incomplete json object as we read
			if _, ok := err.(*json.SyntaxError); ok {
				dec = json.NewDecoder(rdr)
				continue
			}
			// io.ErrUnexpectedEOF is returned from json.Decoder when there is
			// remaining data in the parser's buffer while an io.EOF occurs.
			// If the json logger writes a partial json log entry to the disk
			// while at the same time the decoder tries to decode it, the race condition happens.
			if err == io.ErrUnexpectedEOF {
				reader := io.MultiReader(dec.Buffered(), rdr)
				dec = json.NewDecoder(reader)
				continue
			}
			logrus.WithError(err).WithField("retries", retries).Warn("got error while decoding json")
		}
		return msg, err
	}
}

func getTailReader(ctx context.Context, r SizeReaderAt, req int) (io.Reader, int, error) {
	return tailfile.NewTailReader(ctx, r, req)
}

func newSectionReader(f *os.File) (*io.SectionReader, error) {
	// seek to the end to get the size
	// we'll leave this at the end of the file since section reader does not advance the reader
	size, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, errors.Wrap(err, "error getting current file size")
	}
	return io.NewSectionReader(f, 0, size), nil
}

// ReadLogs decodes entries from log files and sends them the passed in watcher
//
// Note: Using the follow option can become inconsistent in cases with very frequent rotations and max log files is 1.
// TODO: Consider a different implementation which can effectively follow logs under frequent rotations.
func (w *LogFile) ReadLogs(config ReadConfig, watcher *LogWatcher) {
	w.mu.RLock()
	currentFile, err := os.Open(w.f.Name())
	if err != nil {
		w.mu.RUnlock()
		watcher.Err <- err
		return
	}
	defer currentFile.Close()

	currentChunk, err := newSectionReader(currentFile)
	if err != nil {
		w.mu.RUnlock()
		watcher.Err <- err
		return
	}

	if config.Tail != 0 {
		// TODO(@cpuguy83): Instead of opening every file, only get the files which
		// are needed to tail.
		// This is especially costly when compression is enabled.
		files, err := w.openRotatedFiles(config)
		w.mu.RUnlock()
		if err != nil {
			watcher.Err <- err
			return
		}
		closeFiles := func() {
			for _, f := range files {
				f.Close()
				fileName := f.Name()
				if strings.HasSuffix(fileName, tmpLogfileSuffix) {
					err := w.filesRefCounter.Dereference(fileName)
					if err != nil {
						logrus.Errorf("Failed to dereference the log file %q: %v", fileName, err)
					}
				}
			}
		}
		readers := make([]SizeReaderAt, 0, len(files)+1)
		for _, f := range files {
			stat, err := f.Stat()
			if err != nil {
				watcher.Err <- errors.Wrap(err, "error reading size of rotated file")
				closeFiles()
				return
			}
			readers = append(readers, io.NewSectionReader(f, 0, stat.Size()))
		}
		if currentChunk.Size() > 0 {
			readers = append(readers, currentChunk)
		}

		tailFiles(readers, watcher, w.createDecoder, w.getTailReader, config)
		closeFiles()

		w.mu.RLock()
	}

	if !config.Follow || w.closed {
		w.mu.RUnlock()
		return
	}
	w.mu.RUnlock()
	followLogs(currentFile, watcher, w.createDecoder, config.Since, config.Until)
}

func tailFiles(files []SizeReaderAt, watcher *LogWatcher, createDecoder makeDecoderFunc, getTailReader GetTailReaderFunc, config ReadConfig) {
	nLines := config.Tail

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// TODO(@cpuguy83): we should plumb a context through instead of dealing with `WatchClose()` here.
	go func() {
		select {
		case <-ctx.Done():
		case <-watcher.WatchConsumerGone():
			cancel()
		}
	}()

	readers := make([]io.Reader, 0, len(files))

	if config.Tail > 0 {
		for i := len(files) - 1; i >= 0 && nLines > 0; i-- {
			tail, n, err := getTailReader(ctx, files[i], nLines)
			if err != nil {
				watcher.Err <- errors.Wrap(err, "error finding file position to start log tailing")
				return
			}
			nLines -= n
			readers = append([]io.Reader{tail}, readers...)
		}
	} else {
		for _, r := range files {
			readers = append(readers, &wrappedReaderAt{ReaderAt: r})
		}
	}

	rdr := io.MultiReader(readers...)
	decodeLogLine := createDecoder(rdr)
	for {
		msg, err := decodeLogLine()
		if err != nil {
			if errors.Cause(err) != io.EOF {
				watcher.Err <- err
			}
			return
		}
		if !config.Since.IsZero() && msg.Timestamp.Before(config.Since) {
			continue
		}
		if !config.Until.IsZero() && msg.Timestamp.After(config.Until) {
			return
		}
		select {
		case <-ctx.Done():
			return
		case watcher.Msg <- msg:
		}
	}
}

func (w *LogFile) openRotatedFiles(config ReadConfig) (files []*os.File, err error) {
	w.rotateMu.Lock()
	defer w.rotateMu.Unlock()

	defer func() {
		if err == nil {
			return
		}
		for _, f := range files {
			f.Close()
			if strings.HasSuffix(f.Name(), tmpLogfileSuffix) {
				err := os.Remove(f.Name())
				if err != nil && !os.IsNotExist(err) {
					logrus.Warnf("Failed to remove logfile: %v", err)
				}
			}
		}
	}()

	for i := w.maxFiles; i > 1; i-- {
		f, err := os.Open(fmt.Sprintf("%s.%d", w.f.Name(), i-1))
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, errors.Wrap(err, "error opening rotated log file")
			}

			fileName := fmt.Sprintf("%s.%d.gz", w.f.Name(), i-1)
			decompressedFileName := fileName + tmpLogfileSuffix
			tmpFile, err := w.filesRefCounter.GetReference(decompressedFileName, func(refFileName string, exists bool) (*os.File, error) {
				if exists {
					return os.Open(refFileName)
				}
				return decompressfile(fileName, refFileName, config.Since)
			})

			if err != nil {
				if !os.IsNotExist(errors.Cause(err)) {
					return nil, errors.Wrap(err, "error getting reference to decompressed log file")
				}
				continue
			}
			if tmpFile == nil {
				// The log before `config.Since` does not need to read
				break
			}

			files = append(files, tmpFile)
			continue
		}
		files = append(files, f)
	}

	return files, nil
}

func decompressfile(fileName, destFileName string, since time.Time) (*os.File, error) {
	cf, err := os.Open(fileName)
	if err != nil {
		return nil, errors.Wrap(err, "error opening file for decompression")
	}
	defer cf.Close()

	rc, err := gzip.NewReader(cf)
	if err != nil {
		return nil, errors.Wrap(err, "error making gzip reader for compressed log file")
	}
	defer rc.Close()

	// Extract the last log entry timestramp from the gzip header
	extra := &rotateFileMetadata{}
	err = json.Unmarshal(rc.Header.Extra, extra)
	if err == nil && extra.LastTime.Before(since) {
		return nil, nil
	}

	rs, err := os.OpenFile(destFileName, os.O_CREATE|os.O_RDWR, 0640)
	if err != nil {
		return nil, errors.Wrap(err, "error creating file for copying decompressed log stream")
	}

	_, err = pools.Copy(rs, rc)
	if err != nil {
		rs.Close()
		rErr := os.Remove(rs.Name())
		if rErr != nil && !os.IsNotExist(rErr) {
			logrus.Errorf("Failed to remove logfile: %v", rErr)
		}
		return nil, errors.Wrap(err, "error while copying decompressed log stream to file")
	}

	return rs, nil
}

func followLogs(f *os.File, logWatcher *LogWatcher, createDecoder makeDecoderFunc, since, until time.Time) {
	decodeLogLine := createDecoder(f)

	name := f.Name()
	fileWatcher, err := watchFile(name)
	if err != nil {
		logWatcher.Err <- err
		return
	}
	defer func() {
		f.Close()
		fileWatcher.Close()
	}()

	var retries int
	handleRotate := func() error {
		f.Close()
		fileWatcher.Remove(name)

		// retry when the file doesn't exist
		for retries := 0; retries <= 5; retries++ {
			f, err = os.Open(name)
			if err == nil || !os.IsNotExist(err) {
				break
			}
		}
		if err != nil {
			return err
		}
		if err := fileWatcher.Add(name); err != nil {
			return err
		}
		decodeLogLine = createDecoder(f)
		return nil
	}

	errRetry := errors.New("retry")
	errDone := errors.New("done")
	waitRead := func() error {
		select {
		case e := <-fileWatcher.Events():
			switch e.Op {
			case fsnotify.Write:
				decodeLogLine = createDecoder(f)
				return nil
			case fsnotify.Rename, fsnotify.Remove:
				select {
				case <-logWatcher.WatchProducerGone():
					return errDone
				case <-logWatcher.WatchConsumerGone():
					return errDone
				default:
				}
				if err := handleRotate(); err != nil {
					return err
				}
				return nil
			}
			return errRetry
		case err := <-fileWatcher.Errors():
			logrus.Debugf("logger got error watching file: %v", err)
			// Something happened, let's try and stay alive and create a new watcher
			if retries <= 5 {
				fileWatcher.Close()
				fileWatcher, err = watchFile(name)
				if err != nil {
					return err
				}
				retries++
				return errRetry
			}
			return err
		case <-logWatcher.WatchProducerGone():
			return errDone
		case <-logWatcher.WatchConsumerGone():
			return errDone
		}
	}

	handleDecodeErr := func(err error) error {
		if errors.Cause(err) != io.EOF {
			return err
		}

		for {
			err := waitRead()
			if err == nil {
				break
			}
			if err == errRetry {
				continue
			}
			return err
		}
		return nil
	}

	// main loop
	for {
		msg, err := decodeLogLine()
		if err != nil {
			if err := handleDecodeErr(err); err != nil {
				if err == errDone {
					return
				}
				// we got an unrecoverable error, so return
				logWatcher.Err <- err
				return
			}
			// ready to try again
			continue
		}
		retries = 0 // reset retries since we've succeeded
		if !since.IsZero() && msg.Timestamp.Before(since) {
			continue
		}
		if !until.IsZero() && msg.Timestamp.After(until) {
			return
		}
		// send the message, unless the consumer is gone
		select {
		case logWatcher.Msg <- msg:
		case <-logWatcher.WatchConsumerGone():
			return
		}
	}
}

func watchFile(name string) (filenotify.FileWatcher, error) {
	var fileWatcher filenotify.FileWatcher

	if runtime.GOOS == "windows" {
		// FileWatcher on Windows files is based on the syscall notifications which has an issue because of file caching.
		// It is based on ReadDirectoryChangesW() which doesn't detect writes to the cache. It detects writes to disk only.
		// Because of the OS lazy writing, we don't get notifications for file writes and thereby the watcher
		// doesn't work. Hence for Windows we will use poll based notifier.
		fileWatcher = filenotify.NewPollingWatcher()
	} else {
		var err error
		fileWatcher, err = filenotify.New()
		if err != nil {
			return nil, err
		}
	}

	logger := logrus.WithFields(logrus.Fields{
		"module": "logger",
		"file":   name,
	})

	if err := fileWatcher.Add(name); err != nil {
		// we will retry using file poller.
		logger.WithError(err).Warnf("falling back to file poller")
		fileWatcher.Close()
		fileWatcher = filenotify.NewPollingWatcher()

		if err := fileWatcher.Add(name); err != nil {
			fileWatcher.Close()
			logger.WithError(err).Debugf("error watching log file for modifications")
			return nil, err
		}
	}

	return fileWatcher, nil
}

// LogFile is Logger implementation for default Docker logging.
type LogFile struct {
	mu              sync.RWMutex // protects the logfile access
	f               *os.File     // store for closing
	closed          bool
	rotateMu        sync.Mutex // blocks the next rotation until the current rotation is completed
	currentSize     int64      // current size of the latest file
	maxFiles        int        // maximum number of files
	compress        bool       // whether old versions of log files are compressed
	filesRefCounter refCounter // keep reference-counted of decompressed files
	notifyRotate    *pubsub.Publisher
	createDecoder   makeDecoderFunc
	getTailReader   GetTailReaderFunc
	perms           os.FileMode
	logPath         string
}

// Close file close
func (w *LogFile) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return nil
	}
	if err := w.f.Close(); err != nil {
		return err
	}
	w.closed = true
	return nil
}

// ReadConfig read log config
type ReadConfig struct {
	Since  time.Time
	Until  time.Time
	Tail   int
	Follow bool
}

// SizeReaderAt defines a ReaderAt that also reports its size.
// This is used for tailing log files.
type SizeReaderAt interface {
	io.ReaderAt
	Size() int64
}

// NewLogFile creates new LogFile
func NewLogFile(logPath string, maxFiles int, compress bool, decodeFunc makeDecoderFunc, perms os.FileMode, getTailReader GetTailReaderFunc) (*LogFile, error) {
	log, err := os.OpenFile(logPath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, perms)
	if err != nil {
		return nil, err
	}

	size, err := log.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}

	return &LogFile{
		f:               log,
		currentSize:     size,
		maxFiles:        maxFiles,
		compress:        compress,
		filesRefCounter: refCounter{counter: make(map[string]int)},
		notifyRotate:    pubsub.NewPublisher(0, 1),
		createDecoder:   decodeFunc,
		perms:           perms,
		getTailReader:   getTailReader,
		logPath:         logPath,
	}, nil
}
