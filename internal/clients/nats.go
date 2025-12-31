package clients

import (
	"fmt"
	"time"

	"gogin/internal/config"

	"github.com/nats-io/nats.go"
)

// NATSClient wraps the NATS JetStream client
type NATSClient struct {
	conn   *nats.Conn
	js     nats.JetStreamContext
	stream string
}

// NewNATSClient creates a new NATS JetStream client
func NewNATSClient(cfg config.NATSConfig) (*NATSClient, error) {
	opts := []nats.Option{
		nats.Name("goapi"),
		nats.Timeout(10 * time.Second),
		nats.ReconnectWait(2 * time.Second),
		nats.MaxReconnects(-1), // Infinite reconnects
	}

	// Add token if provided
	if cfg.Token != "" {
		opts = append(opts, nats.Token(cfg.Token))
	}

	// Connect to NATS with multiple URLs
	conn, err := nats.Connect(
		formatNATSURLs(cfg.URLs),
		opts...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Create JetStream context
	js, err := conn.JetStream()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	client := &NATSClient{
		conn:   conn,
		js:     js,
		stream: cfg.StreamName,
	}

	// Ensure the stream exists
	if err := client.ensureStream(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to ensure stream: %w", err)
	}

	return client, nil
}

// ensureStream creates the stream if it doesn't exist
func (n *NATSClient) ensureStream() error {
	// Check if stream exists
	_, err := n.js.StreamInfo(n.stream)
	if err == nil {
		return nil // Stream already exists
	}

	// Create stream
	_, err = n.js.AddStream(&nats.StreamConfig{
		Name:     n.stream,
		Subjects: []string{n.stream + ".*"},
		Storage:  nats.FileStorage,
		MaxAge:   7 * 24 * time.Hour, // Keep messages for 7 days
	})

	if err != nil {
		return fmt.Errorf("failed to create stream: %w", err)
	}

	return nil
}

// Publish publishes a message to a subject
func (n *NATSClient) Publish(subject string, data []byte) error {
	fullSubject := n.stream + "." + subject
	_, err := n.js.Publish(fullSubject, data)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}
	return nil
}

// Subscribe creates a durable subscription to a subject
func (n *NATSClient) Subscribe(subject, durableName string, handler nats.MsgHandler) (*nats.Subscription, error) {
	fullSubject := n.stream + "." + subject

	sub, err := n.js.Subscribe(fullSubject, handler, nats.Durable(durableName))
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe: %w", err)
	}

	return sub, nil
}

// QueueSubscribe creates a queue subscription for load balancing
func (n *NATSClient) QueueSubscribe(subject, queue, durableName string, handler nats.MsgHandler) (*nats.Subscription, error) {
	fullSubject := n.stream + "." + subject

	sub, err := n.js.QueueSubscribe(fullSubject, queue, handler, nats.Durable(durableName))
	if err != nil {
		return nil, fmt.Errorf("failed to queue subscribe: %w", err)
	}

	return sub, nil
}

// HealthCheck performs a health check on NATS
func (n *NATSClient) HealthCheck() error {
	if n.conn == nil || !n.conn.IsConnected() {
		return fmt.Errorf("NATS connection is not active")
	}

	// Try to get stream info
	_, err := n.js.StreamInfo(n.stream)
	if err != nil {
		return fmt.Errorf("NATS health check failed: %w", err)
	}

	return nil
}

// Close closes the NATS connection
func (n *NATSClient) Close() {
	if n.conn != nil {
		n.conn.Close()
	}
}

// GetJetStream returns the JetStream context for advanced operations
func (n *NATSClient) GetJetStream() nats.JetStreamContext {
	return n.js
}

// GetConnection returns the underlying NATS connection
func (n *NATSClient) GetConnection() *nats.Conn {
	return n.conn
}

// formatNATSURLs formats multiple NATS URLs into a single string
func formatNATSURLs(urls []string) string {
	if len(urls) == 0 {
		return ""
	}
	if len(urls) == 1 {
		return urls[0]
	}

	result := urls[0]
	for i := 1; i < len(urls); i++ {
		result += "," + urls[i]
	}
	return result
}
