package services

import (
	"fmt"

	"github.com/ethereum/go-ethereum/ethclient"
)

// ClientResolver provides access to chain-specific Ethereum clients
type ClientResolver interface {
	// GetClient returns the ethclient.Client for the specified chain ID
	GetClient(chainID uint64) (*ethclient.Client, error)
}

// SimpleClientResolver is a basic implementation of ClientResolver that maintains a map of chain IDs to clients
type SimpleClientResolver struct {
	clients map[uint64]*ethclient.Client
}

// NewSimpleClientResolver creates a new resolver with the provided map of chain IDs to clients
func NewSimpleClientResolver(clients map[uint64]*ethclient.Client) *SimpleClientResolver {
	return &SimpleClientResolver{
		clients: clients,
	}
}

// GetClient returns the client for the specified chain ID
func (r *SimpleClientResolver) GetClient(chainID uint64) (*ethclient.Client, error) {
	client, ok := r.clients[chainID]
	if !ok {
		return nil, fmt.Errorf("no client found for chain ID %d", chainID)
	}
	return client, nil
}

// CreateSimpleClientResolverFromEthClient creates a new resolver with ethclient.Client instances
func CreateSimpleClientResolverFromEthClient(clients map[uint64]*ethclient.Client) *SimpleClientResolver {
	return NewSimpleClientResolver(clients)
}
