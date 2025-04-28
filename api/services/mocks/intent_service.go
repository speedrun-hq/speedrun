package mocks

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/speedrun-hq/speedrun/api/models"
	"github.com/stretchr/testify/mock"
)

// MockIntentService is a mock implementation of the IntentService
type MockIntentService struct {
	mock.Mock
}

func (m *MockIntentService) GetIntent(ctx context.Context, id string) (*models.Intent, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Intent), args.Error(1)
}

func (m *MockIntentService) ListIntents(ctx context.Context) ([]*models.Intent, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Intent), args.Error(1)
}

func (m *MockIntentService) GetIntentsBySender(ctx context.Context, sender string) ([]*models.Intent, error) {
	args := m.Called(ctx, sender)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Intent), args.Error(1)
}

func (m *MockIntentService) GetIntentsByRecipient(ctx context.Context, recipient string) ([]*models.Intent, error) {
	args := m.Called(ctx, recipient)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Intent), args.Error(1)
}

func (m *MockIntentService) CreateIntent(ctx context.Context, id string, sourceChain uint64, destinationChain uint64, token, amount, recipient, sender, intentFee string) (*models.Intent, error) {
	args := m.Called(ctx, id, sourceChain, destinationChain, token, amount, recipient, sender, intentFee)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Intent), args.Error(1)
}

func (m *MockIntentService) StartListening(ctx context.Context, contractAddress common.Address) error {
	args := m.Called(ctx, contractAddress)
	return args.Error(0)
}

func (m *MockIntentService) UnsubscribeAll() {
	m.Called()
}

func (m *MockIntentService) ActiveGoroutines() int32 {
	args := m.Called()
	return int32(args.Int(0))
}
