package websocket

import (
	"encoding/json"
	"io"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Connection wraps a WebSocket connection and provides helper methods
type Connection struct {
	conn          *websocket.Conn
	mu            sync.Mutex
	writeMu       sync.Mutex
	closeOnce     sync.Once
	currentReader io.Reader // For streaming reads of large messages
}

// NewConnection creates a new WebSocket connection wrapper
func NewConnection(conn *websocket.Conn) *Connection {
	return &Connection{
		conn: conn,
	}
}

// ReadMessage reads a message from the WebSocket connection
func (c *Connection) ReadMessage() (*Message, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	messageType, data, err := c.conn.ReadMessage()
	if err != nil {
		return nil, err
	}

	// Only handle text messages for control plane
	if messageType != websocket.TextMessage {
		return nil, io.EOF
	}

	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}

	return &msg, nil
}

// WriteMessage writes a message to the WebSocket connection
func (c *Connection) WriteMessage(msg *Message) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	return c.conn.WriteMessage(websocket.TextMessage, data)
}

// WriteBinary writes binary data to the WebSocket connection
func (c *Connection) WriteBinary(data []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	return c.conn.WriteMessage(websocket.BinaryMessage, data)
}

// ReadBinary reads binary data from the WebSocket connection
func (c *Connection) ReadBinary() ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	messageType, data, err := c.conn.ReadMessage()
	if err != nil {
		return nil, err
	}

	if messageType != websocket.BinaryMessage {
		return nil, io.EOF
	}

	return data, nil
}

// WritePing writes a ping message to the WebSocket connection
func (c *Connection) WritePing() error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	return c.conn.WriteMessage(websocket.PingMessage, nil)
}

// Close closes the WebSocket connection
func (c *Connection) Close() error {
	var err error
	c.closeOnce.Do(func() {
		err = c.conn.Close()
	})
	return err
}

// RemoteAddr returns the remote address of the connection
func (c *Connection) RemoteAddr() string {
	return c.conn.RemoteAddr().String()
}

// Conn returns the underlying WebSocket connection
func (c *Connection) Conn() *websocket.Conn {
	return c.conn
}

// Read implements io.Reader interface for bidirectional copying
// Uses NextReader to properly handle large messages that may be larger than the read buffer
func (c *Connection) Read(p []byte) (n int, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If we don't have a current reader, get the next message
	if c.currentReader == nil {
		var messageType int
		messageType, c.currentReader, err = c.conn.NextReader()
		if err != nil {
			return 0, err
		}

		// Only handle binary messages for data transfer
		if messageType != websocket.BinaryMessage {
			c.currentReader = nil
			return 0, io.EOF
		}
	}

	// Read from the current message reader
	n, err = c.currentReader.Read(p)

	// If we've finished reading this message, clear the reader
	if err == io.EOF {
		c.currentReader = nil
		// Don't propagate EOF to the caller; just indicate we read 0 bytes
		// The caller will call Read again for the next message
		if n == 0 {
			err = nil
		} else {
			err = nil
		}
	}

	return n, err
}

// Write implements io.Writer interface for bidirectional copying
func (c *Connection) Write(p []byte) (n int, err error) {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	err = c.conn.WriteMessage(websocket.BinaryMessage, p)
	if err != nil {
		return 0, err
	}

	return len(p), nil
}
