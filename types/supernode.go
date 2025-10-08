package types

import (
	supernodetypes "github.com/LumeraProtocol/lumera/x/supernode/v1/types"
)

// SuperNode represents a supernode in the SDK
type SuperNode struct {
	ValidatorAddress string
	IPAddress        string
	State            string
}

// GetLatestIPAddress extracts the IP address with the highest height from IPAddressHistory
func GetLatestIPAddress(history []*supernodetypes.IPAddressHistory) string {
	if len(history) == 0 {
		return ""
	}

	latestIdx := 0
	maxHeight := history[0].Height

	for i := 1; i < len(history); i++ {
		if history[i].Height > maxHeight {
			maxHeight = history[i].Height
			latestIdx = i
		}
	}

	return history[latestIdx].Address
}

// GetLatestState extracts the state with the highest height from SuperNodeStateRecord
func GetLatestState(states []*supernodetypes.SuperNodeStateRecord) *supernodetypes.SuperNodeState {
	if len(states) == 0 {
		return nil
	}

	latestIdx := 0
	maxHeight := states[0].Height

	for i := 1; i < len(states); i++ {
		if states[i].Height > maxHeight {
			maxHeight = states[i].Height
			latestIdx = i
		}
	}

	return &states[latestIdx].State
}

// SuperNodeFromProto converts a proto supernode to SDK supernode
func SuperNodeFromProto(pb *supernodetypes.SuperNode) *SuperNode {
	if pb == nil {
		return nil
	}

	// Extract the IP address with the highest height from PrevIpAddresses
	ipAddress := GetLatestIPAddress(pb.PrevIpAddresses)
	if ipAddress == "" {
		return nil
	}

	// Extract the state with the highest height from States
	latestState := GetLatestState(pb.States)
	if latestState == nil {
		return nil
	}

	return &SuperNode{
		ValidatorAddress: pb.ValidatorAddress,
		IPAddress:        ipAddress,
		State:            latestState.String(),
	}
}
