package types

import (
	"fmt"
	"reflect"
	"time"

	actiontypes "github.com/LumeraProtocol/lumera/x/action/v1/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	proto "github.com/cosmos/gogoproto/proto"
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

// priceToString converts supported price representations to string.
func priceToString(p interface{}) string {
	if p == nil {
		return ""
	}
	if c, ok := p.(*sdk.Coin); ok {
		if c == nil {
			return ""
		}
		return c.String()
	}
	if c, ok := p.(sdk.Coin); ok {
		return c.String()
	}
	if s, ok := p.(string); ok {
		return s
	}
	// As a fallback, use reflection to try a String() method
	rv := reflect.ValueOf(p)
	if rv.IsValid() {
		m := rv.MethodByName("String")
		if m.IsValid() && m.Type().NumIn() == 0 && m.Type().NumOut() == 1 && m.Type().Out(0).Kind() == reflect.String {
			out := m.Call(nil)
			return out[0].String()
		}
	}
	return fmt.Sprint(p)
}

func decodeMetadata(metadataBytes []byte, at actiontypes.ActionType) ActionMetadata {
	if len(metadataBytes) == 0 {
		return nil
	}

	switch at {
	case actiontypes.ActionTypeCascade:
		var pbMeta actiontypes.CascadeMetadata
		if err := proto.Unmarshal(metadataBytes, &pbMeta); err != nil {
			return nil
		}
		return &CascadeMetadata{
			DataHash:   pbMeta.DataHash,
			FileName:   pbMeta.FileName,
			RQIDsIC:    pbMeta.RqIdsIc,
			RQIDsMax:   pbMeta.RqIdsMax,
			RQIDsIDs:   append([]string(nil), pbMeta.RqIdsIds...),
			Signatures: pbMeta.Signatures,
			Public:     pbMeta.Public,
		}
	case actiontypes.ActionTypeSense:
		var pbMeta actiontypes.SenseMetadata
		if err := proto.Unmarshal(metadataBytes, &pbMeta); err != nil {
			return nil
		}
		return &SenseMetadata{
			DataHash:             pbMeta.DataHash,
			CollectionID:         pbMeta.CollectionId,
			GroupID:              pbMeta.GroupId,
			DDAndFingerprintsIC:  pbMeta.DdAndFingerprintsIc,
			DDAndFingerprintsMax: pbMeta.DdAndFingerprintsMax,
			DDAndFingerprintsIDs: append([]string(nil), pbMeta.DdAndFingerprintsIds...),
			Signatures:           pbMeta.Signatures,
		}
	default:
		return nil
	}
}

// ActionFromProto converts a proto action to SDK action
func ActionFromProto(pb *actiontypes.Action) *Action {
	if pb == nil {
		return nil
	}

	return &Action{
		ID:             pb.ActionID,
		Creator:        pb.Creator,
		Type:           ActionType(pb.ActionType.String()),
		State:          ActionState(pb.State.String()),
		Metadata:       decodeMetadata(pb.Metadata, pb.ActionType),
		Price:          priceToString(pb.Price),
		ExpirationTime: time.Unix(pb.ExpirationTime, 0),
		BlockHeight:    pb.BlockHeight,
		SuperNodes:     pb.SuperNodes,
	}
}
