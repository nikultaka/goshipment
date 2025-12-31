package workers

import (
	"log"

	"gogin/internal/clients"
	"gogin/internal/config"
)

// WorkerManager manages background workers
type WorkerManager struct {
	notificationWorker *NotificationWorker
}

// NewWorkerManager creates a new worker manager
func NewWorkerManager(db *clients.Database, nats *clients.NATSClient, cfg *config.Config) *WorkerManager {
	return &WorkerManager{
		notificationWorker: NewNotificationWorker(db, nats, cfg),
	}
}

// Start starts all background workers
func (m *WorkerManager) Start() error {
	log.Println("🚀 Starting background workers...")

	// Start notification worker
	if err := m.notificationWorker.Start(); err != nil {
		return err
	}

	log.Println("✓ All workers started successfully")
	return nil
}

// Stop stops all background workers
func (m *WorkerManager) Stop() {
	log.Println("Stopping background workers...")
	log.Println("Workers stopped")
}
