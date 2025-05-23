package mocks

import (
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/mock"
)

// MockClientResolver is a mock implementation of ClientResolver
type MockClientResolver struct {
	mock.Mock
}

// GetClient mocks the GetClient method
func (m *MockClientResolver) GetClient(chainID uint64) (*ethclient.Client, error) {
	args := m.Called(chainID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ethclient.Client), args.Error(1)
}
