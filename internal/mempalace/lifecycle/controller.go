// Package lifecycle manages Mem Palace session lifecycle based on Twitch stream status.
//
// It polls the Twitch Helix API (by stream ID) every 30 seconds to detect when
// a stream goes live and when it ends. On stream start, it signals the store
// to create a new session. On stream end, it triggers archive of the session.
//
// Events are published via a channel for other components to react:
//   - EventSessionStart: new session created
//   - EventSessionEnd: session ended, archive initiated
package lifecycle

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Soypete/twitch-llm-bot/metrics"
	"github.com/Soypete/twitch-llm-bot/twitch/helix"
)

type SessionEvent struct {
	Type      EventType
	StreamID  string
	StartedAt time.Time
}

type EventType int

const (
	EventSessionStart EventType = iota
	EventSessionEnd
)

type Controller struct {
	helixClient  *helix.Client
	events       chan SessionEvent
	pollInterval time.Duration
	mu           sync.RWMutex
	active       bool
	streamID     string
	startedAt    time.Time
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

func NewController(helixClient *helix.Client, pollInterval time.Duration) *Controller {
	if pollInterval == 0 {
		pollInterval = 30 * time.Second
	}
	return &Controller{
		helixClient:  helixClient,
		events:       make(chan SessionEvent, 10),
		pollInterval: pollInterval,
	}
}

func (c *Controller) Events() <-chan SessionEvent {
	return c.events
}

func (c *Controller) IsActive() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.active
}

func (c *Controller) StreamID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.streamID
}

func (c *Controller) GetStartedAt() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.startedAt
}

func (c *Controller) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.active {
		return fmt.Errorf("lifecycle controller already running")
	}

	ctx, c.cancel = context.WithCancel(ctx)

	if err := c.checkAndAttachExisting(ctx); err != nil {
		return fmt.Errorf("failed to check for existing stream: %w", err)
	}

	c.wg.Add(1)
	go c.poll(ctx)

	return nil
}

func (c *Controller) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.active {
		return nil
	}

	if c.cancel != nil {
		c.cancel()
	}
	c.wg.Wait()

	c.active = false
	c.streamID = ""
	metrics.MempalaceSessionActive.Set(0)

	return nil
}

func (c *Controller) poll(ctx context.Context) {
	defer c.wg.Done()

	ticker := time.NewTicker(c.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.checkStream(ctx)
		}
	}
}

func (c *Controller) checkStream(ctx context.Context) {
	stream, err := c.helixClient.GetBroadcasterStreamStatus(ctx)
	if err != nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if stream != nil && !c.active {
		c.active = true
		c.streamID = stream.ID
		c.startedAt = time.Now()
		metrics.MempalaceSessionActive.Set(1)

		select {
		case c.events <- SessionEvent{
			Type:      EventSessionStart,
			StreamID:  stream.ID,
			StartedAt: c.startedAt,
		}:
		default:
		}
	} else if stream == nil && c.active {
		c.active = false
		metrics.MempalaceSessionActive.Set(0)

		endedStreamID := c.streamID

		select {
		case c.events <- SessionEvent{
			Type:     EventSessionEnd,
			StreamID: endedStreamID,
		}:
		default:
		}

		c.streamID = ""
	}
}

func (c *Controller) checkAndAttachExisting(ctx context.Context) error {
	stream, err := c.helixClient.GetBroadcasterStreamStatus(ctx)
	if err != nil {
		return err
	}

	if stream != nil {
		c.active = true
		c.streamID = stream.ID
		c.startedAt = time.Now()
		metrics.MempalaceSessionActive.Set(1)
	}

	return nil
}
