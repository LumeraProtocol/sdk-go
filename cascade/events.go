package cascade

import (
	"context"

	sdkEvent "github.com/LumeraProtocol/sdk-go/cascade/event"
	snevent "github.com/LumeraProtocol/supernode/v2/sdk/event"
)

// SubscribeToEvents subscribes to specific event types
func (c *Client) SubscribeToEvents(ctx context.Context, eventType sdkEvent.EventType, handler sdkEvent.Handler) error {
	if c.isLocalEventType(eventType) {
		c.addLocalSubscriber(eventType, handler)
		return nil
	}
	bridge := func(ctx context.Context, e snevent.Event) {
		handler(ctx, convertEvent(e))
	}
	return c.snClient.SubscribeToEvents(ctx, snevent.EventType(eventType), bridge)
}

// SubscribeToAllEvents subscribes to all event types
func (c *Client) SubscribeToAllEvents(ctx context.Context, handler sdkEvent.Handler) error {
	c.addLocalAll(handler)
	bridge := func(ctx context.Context, e snevent.Event) {
		handler(ctx, convertEvent(e))
	}
	return c.snClient.SubscribeToAllEvents(ctx, bridge)
}

func convertEvent(e snevent.Event) sdkEvent.Event {
	data := make(sdkEvent.EventData, len(e.Data))
	for k, v := range e.Data {
		data[sdkEvent.EventDataKey(k)] = v
	}
	return sdkEvent.Event{
		Type:      sdkEvent.EventType(e.Type),
		TaskID:    e.TaskID,
		TaskType:  e.TaskType,
		Timestamp: e.Timestamp,
		ActionID:  e.ActionID,
		Data:      data,
	}
}
