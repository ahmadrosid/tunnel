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
	conn         *websocket.Conn
	mu           sync.Mutex
	writeMu      sync.Mutex
	closeOnce    sync.Once
	readBuffer   []byte   // Buffer for partial reads from binary messages
	readOffset   int      // Current offset in readBuffer
	binaryQueue  [][]byte // Queue of binary messages read by ReadMessage()
}

// NewConnection creates a new WebSocket connection wrapper
func NewConnection(conn *websocket.Conn) *Connection {
	return &Connection{
		conn: conn,
	}
}

// ReadMessage reads a message from the WebSocket connection for control plane.
// This method is used by HandleMessages() loop to read JSON control messages.
// Binary messages are queued for Read() to consume, avoiding race conditions.
func (c *Connection) ReadMessage() (*Message, error) {
	for {
		c.mu.Lock()
		messageType, data, err := c.conn.ReadMessage()

		if err != nil {
			c.mu.Unlock()
			return nil, err
		}

		// If it's a binary message, queue it for Read() and continue reading
		if messageType == websocket.BinaryMessage {
			c.binaryQueue = append(c.binaryQueue, data)
			c.mu.Unlock()
			continue
		}

		c.mu.Unlock()

		// Only handle text messages for control plane
		if messageType != websocket.TextMessage {
			continue
		}

		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, err
		}

		return &msg, nil
	}
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
// Reads binary WebSocket messages and buffers them for io.Copy operations
func (c *Connection) Read(p []byte) (n int, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If we have buffered data, return it first
	if c.readOffset < len(c.readBuffer) {
		n = copy(p, c.readBuffer[c.readOffset:])
		c.readOffset += n

		// If we've consumed the entire buffer, clear it
		if c.readOffset >= len(c.readBuffer) {
			c.readBuffer = nil
			c.readOffset = 0
		}

		return n, nil
	}

	// Check if there are queued binary messages from ReadMessage()
	if len(c.binaryQueue) > 0 {
		c.readBuffer = c.binaryQueue[0]
		c.binaryQueue = c.binaryQueue[1:]
		c.readOffset = 0

		// Copy as much as we can to the caller's buffer
		n = copy(p, c.readBuffer)
		c.readOffset = n

		// If we didn't consume everything, keep the buffer for next Read()
		if c.readOffset >= len(c.readBuffer) {
			c.readBuffer = nil
			c.readOffset = 0
		}

		return n, nil
	}

	// No buffered or queued data, read next WebSocket message
	var messageType int
	messageType, c.readBuffer, err = c.conn.ReadMessage()
	if err != nil {
		return 0, err
	}

	// Only handle binary messages for data transfer
	if messageType != websocket.BinaryMessage {
		c.readBuffer = nil
		return 0, io.EOF
	}

	// Copy as much as we can to the caller's buffer
	n = copy(p, c.readBuffer)
	c.readOffset = n

	// If we didn't consume everything, keep the buffer for next Read()
	if c.readOffset >= len(c.readBuffer) {
		c.readBuffer = nil
		c.readOffset = 0
	}

	return n, nil
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
