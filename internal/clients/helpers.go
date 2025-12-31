package clients

import (
	"context"
	"time"
)

// createContext creates a context with timeout
func createContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}
