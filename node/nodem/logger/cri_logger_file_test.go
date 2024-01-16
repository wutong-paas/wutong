package logger

import (
	"context"
	"fmt"
	"testing"
)

func TestReadLogs(t *testing.T) {
	watch := NewLogWatcher()
	go func() {
		for msg := range watch.Msg {
			fmt.Printf("msg is %v", string(msg.Line))
		}
	}()
	ReadLogs(context.Background(), "./test.log", "123", &ReadConfig{Follow: false, Tail: -1}, nil, watch)
}
