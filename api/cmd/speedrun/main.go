package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/speedrun-hq/speedrun/api/clients/evm"
	"github.com/speedrun-hq/speedrun/api/config"
	"github.com/speedrun-hq/speedrun/api/db"
	"github.com/speedrun-hq/speedrun/api/handlers"
	"github.com/speedrun-hq/speedrun/api/logging"
	"github.com/speedrun-hq/speedrun/api/services"
)

const (
	shutdownTimeout = 30 * time.Second
)

func main() {
	flags := parseFlags()
	log := logging.New(os.Stdout, flags.LogLevel, flags.LogJSON)

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load config")
	}

	ctx := context.Background()

	// Initialize database
	log.Info().Msg("Initializing database connection")
	database, err := db.NewPostgresDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize database")
	}

	defer func() {
		if err := database.Close(); err != nil {
			log.Error().Err(err).Msg("Failed to close database")
		}
	}()

	log.Info().Msg("Database connection established successfully")

	// Initialize Ethereum clients
	clients, err := evm.ResolveClientsFromConfig(ctx, *cfg, log)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize Ethereum clients")
	}

	// Create services for all chains
	intentServices, fulfillmentServices, settlementServices, err := createServices(clients, database, cfg, log)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create services")
	}

	// Create metrics service
	metricsService := services.NewMetricsService(log)

	// Register all services with the metrics service
	for chainID, intentService := range intentServices {
		metricsService.RegisterIntentService(chainID, intentService)
	}

	for chainID, fulfillmentService := range fulfillmentServices {
		metricsService.RegisterFulfillmentService(chainID, fulfillmentService)
	}

	for chainID, settlementService := range settlementServices {
		metricsService.RegisterSettlementService(chainID, settlementService)
	}

	// Start the metrics updater
	metricsService.StartMetricsUpdater(ctx)
	log.Info().Msg("Started Prometheus metrics service")

	// Create event catchup service for this chain
	eventCatchupService := services.NewEventCatchupService(
		intentServices,
		fulfillmentServices,
		settlementServices,
		database,
		log,
	)

	// Register EventCatchupService with metrics service
	metricsService.RegisterEventCatchupService(eventCatchupService)

	err = eventCatchupService.StartListening(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to start event catchup service")
	}

	// Start subscription supervisor to monitor and restart services if needed
	eventCatchupService.StartGoroutine("subscription-supervisor", func() {
		eventCatchupService.StartSubscriptionSupervisor(ctx, cfg)
	})
	log.Info().Msg("Started subscription supervisor to monitor service health")

	// Perform a simple diagnostic check on clients
	log.Info().Msg("Performing basic diagnostic checks on clients...")

	// Get the first chain's services for the HTTP server
	firstChainID := uint64(0)
	for chainID := range intentServices {
		firstChainID = chainID
		break
	}
	intentService := intentServices[firstChainID]
	fulfillmentService := fulfillmentServices[firstChainID]

	// Create and start the server
	server := handlers.NewServer(fulfillmentService, intentService, metricsService, database, log)

	// Set up graceful shutdown
	shutdownCtx, shutdownCancel := context.WithCancel(ctx)
	defer shutdownCancel()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		addr := fmt.Sprintf(":%s", cfg.Port)

		if err := server.Start(addr); err != nil {
			log.Error().Err(err).Str("addr", addr).Msg("Server error")

			// Signal shutdown on server error
			shutdownCancel()
		}
	}()

	// Wait for shutdown signal
	select {
	case <-sigChan:
		log.Info().Msg("Shutdown signal received, cleaning up services...")
	case <-shutdownCtx.Done():
		log.Info().Msg("Shutdown context cancelled, cleaning up services...")
	}

	// Shutdown all services gracefully
	var shutdownErrors []error

	// Shutdown event catchup service
	log.Info().Msg("Shutting down event catchup service...")
	if err := eventCatchupService.Shutdown(shutdownTimeout); err != nil {
		err = errors.Wrap(err, "failed to shutdown event catchup service")
		shutdownErrors = append(shutdownErrors, err)
	}

	// Shutdown intent services
	for chainID, intentService := range intentServices {
		log.Info().Uint64(logging.FieldChain, chainID).Msg("Shutting down intent service")
		if err := intentService.Shutdown(shutdownTimeout); err != nil {
			err = errors.Wrap(err, "failed to shutdown intent service")
			shutdownErrors = append(shutdownErrors, err)
		}
	}

	// Shutdown fulfillment services
	for chainID, fulfillmentService := range fulfillmentServices {
		log.Info().Uint64(logging.FieldChain, chainID).Msg("Shutting down fulfillment service")
		if err := fulfillmentService.Shutdown(shutdownTimeout); err != nil {
			err = errors.Wrap(err, "failed to shutdown fulfillment service")
			shutdownErrors = append(shutdownErrors, err)
		}
	}

	// Shutdown settlement services
	for chainID, settlementService := range settlementServices {
		log.Info().Uint64(logging.FieldChain, chainID).Msg("Shutting down settlement service")
		if err := settlementService.Shutdown(shutdownTimeout); err != nil {
			err = errors.Wrap(err, "failed to shutdown settlement service")
			shutdownErrors = append(shutdownErrors, err)
		}
	}

	// Log any shutdown errors
	if len(shutdownErrors) > 0 {
		log.Error().Int("errors_count", len(shutdownErrors)).Msg("Encountered errors during shutdown")
		for _, err := range shutdownErrors {
			log.Error().Err(err).Msg("Error during shutdown")
		}
		return
	}

	log.Info().Msg("All services shut down successfully")
}

// createServices creates and returns the intent and fulfillment services for each chain
func createServices(
	clients map[uint64]*ethclient.Client,
	db db.Database,
	cfg *config.Config,
	logger zerolog.Logger,
) (
	map[uint64]*services.IntentService,
	map[uint64]*services.FulfillmentService,
	map[uint64]*services.SettlementService,
	error,
) {
	intentServices := make(map[uint64]*services.IntentService)
	fulfillmentServices := make(map[uint64]*services.FulfillmentService)
	settlementServices := make(map[uint64]*services.SettlementService)

	// Create a client resolver for cross-chain operations
	clientResolver := services.NewSimpleClientResolver(clients)

	for chainID, client := range clients {
		// Create intent service
		intentService, err := services.NewIntentService(
			client,
			clientResolver,
			db,
			cfg.IntentInitiatedEventABI,
			chainID,
			logger,
		)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to create intent service for chain %d: %v", chainID, err)
		}
		intentServices[chainID] = intentService

		// Create fulfillment service
		fulfillmentService, err := services.NewFulfillmentService(
			client,
			clientResolver,
			db,
			cfg.IntentFulfilledEventABI,
			chainID,
			logger,
		)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to create fulfillment service for chain %d: %v", chainID, err)
		}
		fulfillmentServices[chainID] = fulfillmentService

		// Create settlement service
		settlementService, err := services.NewSettlementService(
			client,
			clientResolver,
			db,
			cfg.IntentSettledEventABI,
			chainID,
			logger,
		)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to create settlement service for chain %d: %v", chainID, err)
		}
		settlementServices[chainID] = settlementService
	}

	return intentServices, fulfillmentServices, settlementServices, nil
}

type flagSet struct {
	LogJSON  bool
	LogLevel zerolog.Level
}

func parseFlags() flagSet {
	var (
		logJSON        bool
		logLevel       string
		logLevelParsed zerolog.Level
	)

	flag.BoolVar(&logJSON, "log-json", false, "Output logs in JSON format")
	flag.StringVar(&logLevel, "log-level", "info", "Set log level (debug, info, warn, error)")

	flag.Parse()

	switch logLevel {
	case "debug":
		logLevelParsed = zerolog.DebugLevel
	case "warn":
		logLevelParsed = zerolog.WarnLevel
	case "error":
		logLevelParsed = zerolog.ErrorLevel
	default:
		logLevelParsed = zerolog.InfoLevel
	}

	return flagSet{
		LogJSON:  logJSON,
		LogLevel: logLevelParsed,
	}
}
