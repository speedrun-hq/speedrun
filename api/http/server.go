package http

import (
	"context"
	"net/http"
	"time"

	"github.com/rs/zerolog"
	"github.com/speedrun-hq/speedrun/api/logging"
)

const shutdownTimeout = 10 * time.Second

// StartAsync starts http server in the background and returns a callback for its shutdown.
func StartAsync(srv *http.Server, logger zerolog.Logger) (shutdownFunc func(context.Context)) {
	logger = logger.With().Str(logging.FieldModule, "http").Logger()

	go func() {
		logger.Info().Msg("Starting HTTP server")

		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			logger.Err(err).Msg("HTTP server error")
		}
	}()

	return func(ctx context.Context) {
		logger.Info().Msg("Shutting down HTTP server")

		ctx, cancel := context.WithTimeout(ctx, shutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			logger.Err(err).Msg("Failed to shutdown HTTP server")
			return
		}

		logger.Info().Msg("HTTP server shutdown complete")
	}
}
