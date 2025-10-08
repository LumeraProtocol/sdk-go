package blockchain

import (
	"strings"

	actiontypes "github.com/LumeraProtocol/lumera/x/action/v1/types"
	"github.com/cosmos/cosmos-sdk/types/query"
)

// QueryOption is a functional option for queries
type QueryOption interface {
	ApplyToActionQuery(*actiontypes.QueryListActionsRequest)
}

type queryOption struct {
	applyToAction func(*actiontypes.QueryListActionsRequest)
}

func (q queryOption) ApplyToActionQuery(req *actiontypes.QueryListActionsRequest) {
	if q.applyToAction != nil {
		q.applyToAction(req)
	}
}

// WithActionType filters by action type enum
func WithActionType(actionType actiontypes.ActionType) QueryOption {
	return queryOption{
		applyToAction: func(req *actiontypes.QueryListActionsRequest) {
			req.ActionType = actionType
		},
	}
}

// WithActionState filters by action state enum
func WithActionState(state actiontypes.ActionState) QueryOption {
	return queryOption{
		applyToAction: func(req *actiontypes.QueryListActionsRequest) {
			req.ActionState = state
		},
	}
}

// WithPagination sets pagination parameters
func WithPagination(limit, offset uint64) QueryOption {
	return queryOption{
		applyToAction: func(req *actiontypes.QueryListActionsRequest) {
			req.Pagination = &query.PageRequest{
				Limit:  limit,
				Offset: offset,
			}
		},
	}
}


// WithActionTypeEnum is an alias for WithActionType for clarity
func WithActionTypeEnum(t actiontypes.ActionType) QueryOption {
	return WithActionType(t)
}

// WithActionTypeStr parses a string into an enum (case-insensitive), falling back to zero value if unknown
func WithActionTypeStr(s string) QueryOption {
	t, ok := parseActionType(s)
	if !ok {
		t = 0 // Zero value (unspecified)
	}
	return WithActionType(t)
}

// WithActionStateEnum is an alias for WithActionState for clarity
func WithActionStateEnum(st actiontypes.ActionState) QueryOption {
	return WithActionState(st)
}

// WithActionStateStr parses a string into an enum (case-insensitive), falling back to zero value if unknown
func WithActionStateStr(s string) QueryOption {
	st, ok := parseActionState(s)
	if !ok {
		st = 0 // Zero value (unspecified)
	}
	return WithActionState(st)
}

// parseActionType attempts to resolve a string to a valid ActionType enum
func parseActionType(s string) (actiontypes.ActionType, bool) {
	if s == "" {
		return 0, false
	}
	if v, ok := actiontypes.ActionType_value[strings.ToUpper(s)]; ok {
		return actiontypes.ActionType(v), true
	}
	return 0, false
}

// parseActionState attempts to resolve a string to a valid ActionState enum
func parseActionState(s string) (actiontypes.ActionState, bool) {
	if s == "" {
		return 0, false
	}
	if v, ok := actiontypes.ActionState_value[strings.ToUpper(s)]; ok {
		return actiontypes.ActionState(v), true
	}
	return 0, false
}