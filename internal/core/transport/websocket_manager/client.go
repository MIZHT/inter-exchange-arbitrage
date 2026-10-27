// Package websocket_manager provides the exchange-agnostic WebSocket transport.
// Exchange protocol details such as subscription messages and payload parsing
// belong in the corresponding exchange package.
package websocket_manager

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	conn      *websocket.Conn
	writeMu   sync.Mutex
	closeOnce sync.Once
	done      chan struct{}
}

func Dial(ctx context.Context, endpoint, proxyURL string, compression ...bool) (*Client, error) {
	dialer := *websocket.DefaultDialer
	dialer.HandshakeTimeout = 10 * time.Second
	if len(compression) > 0 {
		dialer.EnableCompression = compression[0]
	}
	if proxyURL != "" {
		parsed, err := url.Parse(proxyURL)
		if err != nil {
			return nil, fmt.Errorf("parse WebSocket proxy: %w", err)
		}
		dialer.Proxy = http.ProxyURL(parsed)
	}
	conn, _, err := dialer.DialContext(ctx, endpoint, nil)
	if err != nil {
		return nil, err
	}
	client := &Client{conn: conn, done: make(chan struct{})}
	go func() {
		select {
		case <-ctx.Done():
			_ = client.Close()
		case <-client.done:
		}
	}()
	return client, nil
}

func (c *Client) ReadMessage() ([]byte, error) {
	_, payload, err := c.conn.ReadMessage()
	return payload, err
}

func (c *Client) WriteJSON(value any) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.conn.WriteJSON(value)
}

func (c *Client) WriteMessage(messageType int, payload []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.conn.WriteMessage(messageType, payload)
}

func (c *Client) Close() error {
	var err error
	c.closeOnce.Do(func() {
		close(c.done)
		err = c.conn.Close()
	})
	return err
}
