package streamlog

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"net"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

var errNoConnect = errors.New("no connect")
var errClosed = errors.New("conn is closed")
var errCreate = errors.New("conn is not tcp conn")

// Client stream client
type Client struct {
	writer    *bufio.Writer
	conn      *net.TCPConn
	server    string
	closeFlag int32
}

// NewClient 新建客户端
func NewClient(server string) (c *Client, err error) {
	conn, err := net.Dial("tcp", server)
	if err != nil {
		logrus.Error("log stream server connect error.", err.Error())
		c = &Client{
			server: server,
		}
		atomic.StoreInt32(&c.closeFlag, 1)
	}
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		c = &Client{
			conn:   tcpConn,
			writer: bufio.NewWriter(conn),
			server: server,
		}
		c.conn.CloseRead()
		c.conn.SetKeepAlive(true)
	}
	if c == nil {
		return nil, errCreate
	}
	return c, nil
}

// Dial 连接
func (c *Client) Dial() error {
	if c.IsClosed() {
		conn, err := net.Dial("tcp", c.server)
		if err != nil {
			return err
		}
		if tcpConn, ok := conn.(*net.TCPConn); ok {
			c.conn = tcpConn
			c.conn.SetWriteBuffer(1024 * 1024 * 24)
		} else {
			return errCreate
		}
		c.writer = bufio.NewWriter(conn)
		atomic.StoreInt32(&c.closeFlag, 0)
	}
	return nil
}

// ChangeAddress 更换服务地址
func (c *Client) ChangeAddress(server string) error {
	c.server = server
	return c.ReConnect()
}

// ReConnect 重连
func (c *Client) ReConnect() error {
	if !c.IsClosed() {
		c.Close()
	}
	return c.Dial()
}

// Close close
func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.writer = nil
	atomic.StoreInt32(&c.closeFlag, 1)
}

// IsClosed close
func (c *Client) IsClosed() bool {
	return atomic.LoadInt32(&c.closeFlag) == 1
}

// write if get a error. will close the conn.
func (c *Client) Write(message string) error {
	if message == "" {
		return nil
	}
	if c.writer == nil {
		return errNoConnect
	}
	msg, err := c.encode(message)
	if err != nil {
		return err
	}
	err = c.write(string(msg))
	if err != nil {
		c.Close()
	}
	return err
}

func (c *Client) write(message string) error {
	if c.conn == nil {
		return errClosed
	}
	err := c.conn.SetWriteDeadline(time.Now().Add(time.Second * 5))
	if err != nil {
		return err
	}
	if c.writer != nil {
		wrote, err := c.writer.WriteString(message)
		ferr := c.writer.Flush()
		if wrote < len(message) {
			return err
		}
		if ferr != nil {
			return ferr
		}

		return err
	}
	return errClosed
}

// Encode 编码
func (c *Client) encode(message string) ([]byte, error) {
	// 读取消息的长度
	var length = int32(len(message))
	var pkg = new(bytes.Buffer)
	// 写入消息头
	err := binary.Write(pkg, binary.LittleEndian, length)
	if err != nil {
		return nil, err
	}
	// 写入消息实体
	err = binary.Write(pkg, binary.LittleEndian, []byte(message))
	if err != nil {
		return nil, err
	}
	return pkg.Bytes(), nil
}
