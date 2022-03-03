package model

import "github.com/wutong-paas/wutong/cmd/gateway/option"

// Stream -
type Stream struct {
	StreamPort int
}

// NewStream creates a new stream.
func NewStream(conf *option.Config) *Stream {
	return &Stream{
		StreamPort: conf.ListenPorts.Stream,
	}
}
