package blockchain

import (
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

// WithActionType filters by action type
func WithActionType(actionType string) QueryOption {
	return queryOption{
		applyToAction: func(req *actiontypes.QueryListActionsRequest) {
			req.ActionType = actionType
		},
	}
}

// WithActionState filters by action state
func WithActionState(state string) QueryOption {
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
