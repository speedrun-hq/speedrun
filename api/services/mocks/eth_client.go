package mocks

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/mock"
)

// MockEthClient implements the EthClientInterface for testing
type MockEthClient struct {
	mock.Mock
}

// BlockNumber mocks the BlockNumber method
func (m *MockEthClient) BlockNumber(ctx context.Context) (uint64, error) {
	args := m.Called(ctx)
	return args.Get(0).(uint64), args.Error(1)
}

// TransactionByHash mocks the TransactionByHash method
func (m *MockEthClient) TransactionByHash(ctx context.Context, txHash common.Hash) (*types.Transaction, bool, error) {
	args := m.Called(ctx, txHash)
	if args.Get(0) == nil {
		return nil, args.Bool(1), args.Error(2)
	}
	return args.Get(0).(*types.Transaction), args.Bool(1), args.Error(2)
}

// HeaderByNumber mocks the HeaderByNumber method
func (m *MockEthClient) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	args := m.Called(ctx, number)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Header), args.Error(1)
}

// TransactionReceipt mocks the TransactionReceipt method
func (m *MockEthClient) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	args := m.Called(ctx, txHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Receipt), args.Error(1)
}

// BlockByNumber mocks the BlockByNumber method
func (m *MockEthClient) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	args := m.Called(ctx, number)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Block), args.Error(1)
}

// SubscribeFilterLogs mocks the SubscribeFilterLogs method
func (m *MockEthClient) SubscribeFilterLogs(
	ctx context.Context,
	q ethereum.FilterQuery,
	ch chan<- types.Log,
) (ethereum.Subscription, error) {
	args := m.Called(ctx, q, ch)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(ethereum.Subscription), args.Error(1)
}

// FilterLogs mocks the FilterLogs method
func (m *MockEthClient) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	args := m.Called(ctx, q)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.Log), args.Error(1)
}

// BalanceAt mocks the BalanceAt method
func (m *MockEthClient) BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	args := m.Called(ctx, account, blockNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*big.Int), args.Error(1)
}

// CodeAt mocks the CodeAt method
func (m *MockEthClient) CodeAt(ctx context.Context, account common.Address, blockNumber *big.Int) ([]byte, error) {
	args := m.Called(ctx, account, blockNumber)
	return args.Get(0).([]byte), args.Error(1)
}

// CallContract mocks the CallContract method
func (m *MockEthClient) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	args := m.Called(ctx, call, blockNumber)
	return args.Get(0).([]byte), args.Error(1)
}
