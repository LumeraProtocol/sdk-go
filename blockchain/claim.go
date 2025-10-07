package blockchain

import (
	claimtypes "github.com/LumeraProtocol/lumera/x/claim/types"
)

// ClaimClient provides claim module operations
type ClaimClient struct {
	query claimtypes.QueryClient
}

// Add claim-specific methods here as needed
