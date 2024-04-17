package util

import (
	"fmt"
	"math/rand"
	"time"
)

func GenerateMACAddress() string {
	nt := time.Now().UnixNano()
	fmt.Printf("nt: %v\n", nt)
	r := rand.New(rand.NewSource(nt))
	buf := make([]byte, 6)
	_, err := r.Read(buf)
	if err != nil {
		panic(err)
	}

	// 设置第二个字节的最低两位为零，以确保是单播和全局唯一的地址
	buf[0] &^= 0x01
	buf[0] |= 0x02

	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", buf[0], buf[1], buf[2], buf[3], buf[4], buf[5])
}
