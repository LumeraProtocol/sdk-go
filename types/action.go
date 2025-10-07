package types

import (
	"time"

	actiontypes "github.com/LumeraProtocol/lumera/x/action/v1/types"
)

// Action represents an action in the SDK
type Action struct {
	ID             string
	Creator        string
	Type           ActionType
	State          ActionState
	Metadata       ActionMetadata
	Price          string
	ExpirationTime time.Time
	BlockHeight    int64
	SuperNodes     []string
}

// ActionType represents the type of action
type ActionType string

const (
	ActionTypeCascade ActionType = "CASCADE"
	ActionTypeSense   ActionType = "SENSE"
)

// ActionState represents the state of an action
type ActionState string

const (
	ActionStatePending    ActionState = "PENDING"
	ActionStateProcessing ActionState = "PROCESSING"
	ActionStateDone       ActionState = "DONE"
	ActionStateApproved   ActionState = "APPROVED"
	ActionStateFailed     ActionState = "FAILED"
	ActionStateExpired    ActionState = "EXPIRED"
)

// ActionMetadata is an interface for different action metadata types
type ActionMetadata interface {
	Type() ActionType
}

// CascadeMetadata contains cascade-specific metadata
type CascadeMetadata struct {
	DataHash   string
	FileName   string
	RQIDsIC    uint64
	RQIDsMax   uint64
	RQIDsIDs   []string
	Signatures string
	Public     bool
}

func (m *CascadeMetadata) Type() ActionType {
	return ActionTypeCascade
}

// SenseMetadata contains sense-specific metadata
type SenseMetadata struct {
	DataHash             string
	CollectionID         string
	GroupID              string
	DDAndFingerprintsIC  uint64
	DDAndFingerprintsMax uint64
	DDAndFingerprintsIDs []string
	Signatures           string
}

func (m *SenseMetadata) Type() ActionType {
	return ActionTypeSense
}

// ActionFromProto converts a proto action to SDK action
func ActionFromProto(pb *actiontypes.Action) *Action {
	if pb == nil {
		return nil
	}

	return &Action{
		ID:      pb.ActionID,
		Creator: pb.Creator,
		Type:    ActionType(pb.ActionType.String()),
		State:   ActionState(pb.State.String()),
		// Metadata:       decodeMetadata(pb.Metadata, pb.ActionType),
		Price:          pb.Price,
		ExpirationTime: time.Unix(pb.ExpirationTime, 0),
		BlockHeight:    pb.BlockHeight,
		SuperNodes:     pb.SuperNodes,
	}
}
