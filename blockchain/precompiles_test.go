package blockchain

import (
	"context"
	"testing"

	"github.com/LumeraProtocol/sdk-go/pkg/evm/precompiles"
	"github.com/stretchr/testify/require"
)

func TestPrecompileClient_Address(t *testing.T) {
	p := &PrecompileClient{address: precompiles.ActionAddress, abi: precompiles.ActionABI}
	require.Equal(t, precompiles.ActionAddress, p.Address())
	require.NotEmpty(t, p.ABI().Methods)
}

func TestPrecompileClient_CallWithoutEVMClient(t *testing.T) {
	p := &PrecompileClient{address: precompiles.ActionAddress, abi: precompiles.ActionABI}
	_, err := p.Call(context.Background(), "getParams")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not wired")
}

func TestPrecompileClient_SendWithoutEVMClient(t *testing.T) {
	p := &PrecompileClient{address: precompiles.ActionAddress, abi: precompiles.ActionABI}
	_, err := p.Send(context.Background(), "approveAction", nil, "some-action-id")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not wired")
}

func TestPrecompileClient_RejectsUnknownMethod(t *testing.T) {
	p := &PrecompileClient{
		address: precompiles.ActionAddress,
		abi:     precompiles.ActionABI,
		evm:     &EVMClient{},
	}
	_, err := p.Call(context.Background(), "thisMethodDoesNotExist")
	require.Error(t, err)
}
