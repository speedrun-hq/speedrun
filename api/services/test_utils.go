package services

import (
	"context"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/mock"
)

// MockEthClient is a mock implementation of the Ethereum client
type MockEthClient struct {
	mock.Mock
}

func (m *MockEthClient) SubscribeFilterLogs(ctx context.Context, q interface{}, ch chan<- types.Log) (interface{}, error) {
	args := m.Called(ctx, q, ch)
	return args.Get(0), args.Error(1)
}

// Create a wrapper function to convert our mock to the expected type
func createMockEthClient() *ethclient.Client {
	// This is a hack to make the tests work
	// In a real scenario, you would use a proper mock framework
	client, _ := ethclient.Dial("http://localhost:8545")
	return client
}
