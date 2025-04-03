package handlers

import (
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/zeta-chain/zetafast/api/config"
	"github.com/zeta-chain/zetafast/api/db"
	"github.com/zeta-chain/zetafast/api/services"
)

// InitHandlers initializes the handlers with required dependencies
func InitHandlers(clients map[uint64]*ethclient.Client, contractAddresses map[uint64]string, database db.DBInterface) error {
	var err error
	// TODO: Get contract ABI from a configuration or file
	contractABI := `[{"anonymous":false,"inputs":[{"indexed":true,"internalType":"bytes32","name":"intentId","type":"bytes32"},{"indexed":true,"internalType":"address","name":"asset","type":"address"},{"indexed":false,"internalType":"uint256","name":"amount","type":"uint256"},{"indexed":true,"internalType":"address","name":"receiver","type":"address"}],"name":"IntentFulfilled","type":"event"}]`

	// Load config to get default blocks
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	// Create default blocks map from config
	defaultBlocks := make(map[uint64]uint64)
	for chainID, chainConfig := range cfg.ChainConfigs {
		defaultBlocks[chainID] = chainConfig.DefaultBlock
	}

	fulfillmentService, err = services.NewFulfillmentService(clients, contractAddresses, database, contractABI, defaultBlocks)
	return err
}
