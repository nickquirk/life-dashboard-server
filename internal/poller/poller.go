package poller

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/nickquirk/life-dashboard-server/internal/service"
	"gorm.io/gorm"
)

type Poller struct {
	service  service.Service
	db       *gorm.DB
	interval time.Duration
	running  bool
	stopCh   chan struct{}
	wg       sync.WaitGroup
	mu       sync.Mutex
}

// New creates a new Poller instance
func New(svc service.Service, db *gorm.DB) *Poller {
	// define interval for polling
	interval := 5 * time.Minute
	envInterval := os.Getenv("GOOGLE_POLL_INTERVAL")
	if envInterval != "" {
		parsed, err := time.ParseDuration(envInterval)
		if err != nil {
			log.Printf("[Poller] Invalid GOOGLE_POLL_INTERVAL '%s': %v. Using default %v\n", envInterval, err, interval)
			parsed = interval // Reset to default
		}
		interval = parsed
	}

	return &Poller{
		service:  svc,
		db:       db,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// begins background polling goroutine
func (p *Poller) Start() {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return
	}
	p.running = true
	p.mu.Unlock()

	p.wg.Add(1)
	go p.run()
	log.Printf("[Poller] Started with interval: %v\n", p.interval)
}

func (p *Poller) Stop() {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return
	}
	p.running = false
	p.mu.Unlock()

	close(p.stopCh)
	p.wg.Wait()
	log.Println("[Poller] Stopped")
}

func (p *Poller) run() {
	// ensure the waitgroup decrements even if goroutine exits with a panic
	defer p.wg.Done()

	// initial sync after brief startup delay
	select {
	case <-time.After(30 * time.Second):
		p.syncAllUsers()
	case <-p.stopCh:
		return
	}

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.syncAllUsers()
		case <-p.stopCh:
			return
		}
	}
}

func (p *Poller) syncAllUsers() {
	log.Println("[Poller] Starting sync...")

	var userIDs []uint
	err := p.db.Table("users").
		Where("refresh_token IS NOT NULL AND refresh_token != ''").
		Pluck("id", &userIDs).Error
	if err != nil {
		log.Printf("[Poller] Error fetching users: %v\n", err)
		return
	}

	var success, errors int
	for _, userID := range userIDs {
		err := p.syncUser(userID)
		if err != nil {
			log.Printf("[Poller] Error syncing user %d: %v\n", userID, err)
			errors++
			continue
		}
		success++
	}
	log.Printf("[Poller] Complete. Success: %d, Errors: %d\n", success, errors)
}

func (p *Poller) syncUser(userID uint) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	err := p.service.SyncTaskLists(ctx, userID)
	if err != nil {
		return fmt.Errorf("sync task lists: %w", err)
	}

	lists, err := p.service.GetTaskLists(userID)
	if err != nil {
		return fmt.Errorf("get task lists: %w", err)
	}

	for _, list := range lists {
		err := p.service.SyncTasks(ctx, userID, list.ID)
		if err != nil {
			log.Printf("[Poller] Error syncing list %s: %v\n", list.ID, err)
		}
	}
	return nil
}
