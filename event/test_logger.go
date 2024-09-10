package event

import (
	"fmt"
	"os"
	"time"
)

// GetTestLogger GetTestLogger
func GetTestLogger() Logger {
	return &testLogger{}
}

type testLogger struct {
}

func (l *testLogger) GetChan() chan []byte {
	return nil
}

func (l *testLogger) SetChan(ch chan []byte) {

}

func (l *testLogger) Event() string {
	return "test"
}

func (l *testLogger) CreateTime() time.Time {
	return time.Now()
}

func (l *testLogger) Info(message string, info map[string]string) {
	fmt.Println("info:", message)
}

func (l *testLogger) Error(message string, info map[string]string) {
	fmt.Println("error:", message)
}

func (l *testLogger) Debug(message string, info map[string]string) {
	fmt.Println("debug:", message)
}

type testLoggerWriter struct {
}

func (l *testLoggerWriter) SetFormat(f map[string]interface{}) {

}

func (l *testLoggerWriter) Write(b []byte) (n int, err error) {
	return os.Stdout.Write(b)
}

func (l *testLogger) GetWriter(step, level string) LoggerWriter {
	return &testLoggerWriter{}
}
