package evm

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/speedrun-hq/speedrun/api/config"
	"github.com/speedrun-hq/speedrun/api/logging"
	"golang.org/x/sync/errgroup"
)

// ResolveClientsFromConfig provisions a map of [chainID] => ethclient.Client based on the config.
func ResolveClientsFromConfig(
	ctx context.Context,
	cfg config.Config,
	logger zerolog.Logger,
) (map[uint64]*ethclient.Client, error) {
	var (
		clients             = make(map[uint64]*ethclient.Client, len(cfg.ChainConfigs))
		mu                  = sync.Mutex{}
		errGroup, ctxShared = errgroup.WithContext(ctx)
	)

	for chainID := range cfg.ChainConfigs {
		chain := *cfg.ChainConfigs[chainID]
		errGroup.Go(func() error {
			client, err := NewFromConfig(ctxShared, chain, logger)
			if err != nil {
				return errors.Wrapf(err, "failed to create client for chain %d", chain.ChainID)
			}

			mu.Lock()
			clients[chain.ChainID] = client
			mu.Unlock()

			return nil
		})
	}

	if err := errGroup.Wait(); err != nil {
		return nil, err
	}

	return clients, nil
}

// NewFromConfig creates a new ethclient.Client from a chain configuration.
func NewFromConfig(
	ctx context.Context,
	chain config.ChainConfig,
	logger zerolog.Logger,
) (*ethclient.Client, error) {
	logger = logger.With().
		Uint64(logging.FieldChain, chain.ChainID).
		Str(logging.FieldModule, "evm_client").
		Logger()

	isWebSocket := isWebSocketURL(chain.RPCURL)

	var evmClient *ethclient.Client

	switch {
	case chain.ChainID == config.ZetachainMainnetChainID && isWebSocket:
		logger.Info().Msg("For ZetaChain, forcing HTTP connection instead of WebSocket")

		// Convert WebSocket URL to HTTP if necessary
		httpURL := chain.RPCURL
		httpURL = strings.Replace(httpURL, "wss://", "https://", 1)
		httpURL = strings.Replace(httpURL, "ws://", "http://", 1)

		client, err := ethclient.Dial(httpURL)
		if err != nil {
			return nil, errors.Wrap(err, "failed to connect to ZetaChain with HTTP")
		}

		evmClient = client
		isWebSocket = false
	case chain.ChainID == config.ZetachainMainnetChainID:
		client, err := ethclient.Dial(chain.RPCURL)
		if err != nil {
			return nil, errors.Wrap(err, "failed to connect to ZetaChain")
		}

		evmClient = client
	case isWebSocket:
		rpcClient, err := rpc.DialWebsocket(ctx, chain.RPCURL, "")
		if err != nil {
			return nil, errors.Wrap(err, "failed to create WebSocket RPC client")
		}

		evmClient = ethclient.NewClient(rpcClient)

		if err := verifyWebsocketSubscription(ctx, evmClient, logger); err != nil {
			return nil, errors.Wrap(err, "failed to verify WebSocket subscription")
		}

		logger.Info().Msg("Successfully created WebSocket client")
	default:
		logger.Warn().Msg("Using HTTP RPC. Real-time subscriptions may not work. Consider using WebSockets")
		client, err := ethclient.Dial(chain.RPCURL)
		if err != nil {
			return nil, errors.Wrap(err, "failed to connect to chain")
		}

		evmClient = client
	}

	// should not happen
	if evmClient == nil {
		return nil, errors.New("evmClient is nil")
	}

	// verify that the client works
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	bn, err := evmClient.BlockNumber(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get block number")
	}

	logger.Info().
		Bool("is_websocket", isWebSocket).
		Uint64(logging.FieldBlock, bn).
		Msg("Successfully created EVM client")

	return evmClient, nil
}

// verifyWebsocketSubscription tests if a client supports subscriptions by attempting to subscribe to new heads
func verifyWebsocketSubscription(
	ctx context.Context,
	client *ethclient.Client,
	logger zerolog.Logger,
) error {
	logger = logger.With().
		Str(logging.FieldModule, "evm_client_ws_subscription_test").
		Logger()

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// Create a channel to receive headers
	headers := make(chan *types.Header)

	// Try to subscribe to new heads - this only works with websocket connections
	sub, err := client.SubscribeNewHead(ctx, headers)
	if err != nil {
		return errors.Wrap(err, "subscription test failed")
	}

	// Create a channel to signal when we've received a header or timed out
	received := make(chan error, 1)

	defer func() {
		close(received)
		sub.Unsubscribe()
	}()

	go func() {
		select {
		case header := <-headers:
			logger.Info().
				Uint64(logging.FieldBlock, header.Number.Uint64()).
				Str("block_hash", header.Hash().Hex()).
				Msg("Received new block header")

			received <- nil
			return
		case err := <-sub.Err():
			received <- errors.Wrap(err, "subscription error")
			return
		case <-ctx.Done():
			received <- ctx.Err()
			return
		}
	}()

	if err := <-received; err != nil {
		return errors.Wrap(err, "failed to verify WebSocket subscription")
	}

	return nil
}

func isWebSocketURL(url string) bool {
	return strings.HasPrefix(url, "wss://") || strings.HasPrefix(url, "ws://")
}
