package cascade

import (
	"context"

	"github.com/LumeraProtocol/supernode/v2/sdk/event"
)

// SubscribeToEvents subscribes to specific event types
func (c *Client) SubscribeToEvents(ctx context.Context, eventType event.EventType, handler event.Handler) error {
	return c.snClient.SubscribeToEvents(ctx, eventType, handler)
}

// SubscribeToAllEvents subscribes to all event types
func (c *Client) SubscribeToAllEvents(ctx context.Context, handler event.Handler) error {
	return c.snClient.SubscribeToAllEvents(ctx, handler)
}