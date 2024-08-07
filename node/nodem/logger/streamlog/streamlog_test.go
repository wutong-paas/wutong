package streamlog

import (
	"sync"
	"testing"
	"time"

	"github.com/wutong-paas/wutong/node/nodem/logger"

	"github.com/google/uuid"
)

func TestStreamLog(t *testing.T) {
	var wait sync.WaitGroup
	for j := 0; j < 1000; j++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			log, err := New(logger.Info{
				ContainerID:  uuid.New().String(),
				ContainerEnv: []string{"WT_TENANT_ID=" + uuid.New().String(), "WT_SERVICE_ID=" + uuid.New().String()},
				Config:       map[string]string{"stream-server": "192.168.2.203:6362"},
			})
			if err != nil {
				t.Error(err)
				return
			}
			defer log.Close()
			for i := 0; i < 500000; i++ {
				err := log.Log(&logger.Message{
					Line:      []byte("hello word!hello word!hello word!hello word!hello word!hello word!asdasfmaksmfkasmfkamsmakmskamsdaskdaksdmaksmdkamsdkamsdkmaksdmaksdmkamsdkamsdkaksdakdmklamdlkamdsklmalksdmlkamsdlkamdlkamsdlkmalksmdlkadmlkam"),
					Timestamp: time.Now(),
					Source:    "stdout",
				})
				if err != nil {
					t.Error(err)
				}
				time.Sleep(time.Millisecond * 2)
			}
		}()
	}
	wait.Wait()
}

func BenchmarkStreamLog(t *testing.B) {
	log, err := New(logger.Info{
		ContainerID:  uuid.New().String(),
		ContainerEnv: []string{"WT_TENANT_ID=" + uuid.New().String(), "WT_SERVICE_ID=" + uuid.New().String()},
		Config:       map[string]string{"stream-server": "127.0.0.1:5031"},
	})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < t.N; i++ {
		log.Log(&logger.Message{
			Line:      []byte("hello word"),
			Timestamp: time.Now(),
			Source:    "stdout",
		})
	}
}
