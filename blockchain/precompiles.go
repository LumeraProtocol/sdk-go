package blockchain

import (
	"context"
	"fmt"

	"github.com/LumeraProtocol/sdk-go/pkg/evm/precompiles"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// PrecompileClient wraps a Lumera precompile (action, supernode, or wasm) so
// callers can invoke any of its methods without packing calldata manually.
//
// Use Call for read-only methods (forwarded through EthCall) and Send for
// state-changing methods (signed and broadcast as a MsgEthereumTx).
type PrecompileClient struct {
	address common.Address
	abi     abi.ABI
	evm     *EVMClient
}

// Address returns the precompile's fixed Ethereum address.
func (p *PrecompileClient) Address() common.Address { return p.address }

// ABI returns the parsed ABI so callers can introspect method signatures.
func (p *PrecompileClient) ABI() abi.ABI { return p.abi }

// Call invokes a read-only precompile method and returns the unpacked
// outputs in the order the Solidity interface declares them.
func (p *PrecompileClient) Call(ctx context.Context, method string, args ...any) ([]any, error) {
	if p.evm == nil {
		return nil, fmt.Errorf("precompile not wired to an EVMClient")
	}
	data, err := precompiles.PackCall(p.abi, method, args...)
	if err != nil {
		return nil, err
	}
	ret, err := p.evm.CallContract(ctx, p.address, data)
	if err != nil {
		return nil, err
	}
	return precompiles.UnpackReturn(p.abi, method, ret)
}

// Send signs and broadcasts a state-changing precompile call. opts may be
// nil; nonce/gas/fees are resolved from chain state when unset.
func (p *PrecompileClient) Send(ctx context.Context, method string, opts *EthereumTxOptions, args ...any) (*EthereumTransactionResult, error) {
	if p.evm == nil {
		return nil, fmt.Errorf("precompile not wired to an EVMClient")
	}
	data, err := precompiles.PackCall(p.abi, method, args...)
	if err != nil {
		return nil, err
	}
	addr := p.address
	return p.evm.SendEthereumTransaction(ctx, &addr, data, opts)
}
