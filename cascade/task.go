package cascade

import (
	"context"
	"fmt"
	"time"

	snsdk "github.com/LumeraProtocol/supernode/v2/sdk/action"
)

// TaskManager manages task lifecycle
type TaskManager struct {
	client snsdk.Client
}

// NewTaskManager creates a new task manager
func NewTaskManager(client snsdk.Client) *TaskManager {
	return &TaskManager{
		client: client,
	}
}

// TaskInfo contains task information
type TaskInfo struct {
	TaskID string
}

// Wait waits for a task to complete
func (tm *TaskManager) Wait(ctx context.Context, taskID string) (*TaskInfo, error) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			entry, found := tm.client.GetTask(ctx, taskID)
			if !found || entry == nil {
				// Task isn't found yet; continue polling until ctx timeout
				continue
			}

			switch string(entry.Status) {
			case "COMPLETED":
				return &TaskInfo{
					TaskID: taskID,
				}, nil
			case "FAILED":
				if entry.Error != nil {
					return nil, fmt.Errorf("task %s failed: %v (tx: %s)", taskID, entry.Error, entry.TxHash)
				}
				return nil, fmt.Errorf("task %s failed (tx: %s)", taskID, entry.TxHash)
			case "ACTIVE", "PENDING":
				// Keep waiting
				continue
			default:
				// Unknown status; keep waiting conservatively
				continue
			}
		}
	}
}
