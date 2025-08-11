package http

import (
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

const slowRequestThreshold = 500 * time.Millisecond

func Zerolog(log zerolog.Logger, level zerolog.Level) gin.HandlerFunc {
	logFunc := log.Info
	if level == zerolog.DebugLevel {
		logFunc = log.Debug
	}

	return func(c *gin.Context) {
		start := time.Now()

		// process request
		c.Next()

		latency := time.Since(start)

		if latency > slowRequestThreshold {
			logRequest(log.Warn(), c, latency).Msg("SLOW HTTP request")
			return
		}

		logRequest(logFunc(), c, latency).Msg("HTTP request")
	}
}

func logRequest(e *zerolog.Event, c *gin.Context, latency time.Duration) *zerolog.Event {
	return e.
		Str("http.client_ip", c.ClientIP()).
		Str("http.method", c.Request.Method).
		Str("http.path", c.Request.URL.Path).
		Int("http.status", c.Writer.Status()).
		Dur("http.latency", latency).
		Str("http.ua", c.Request.UserAgent())
}

// CORS. Allowed origins should be comma separated. Empty string is treated as `*` wildcard.
func CORS(allowedOrigins string) gin.HandlerFunc {
	if allowedOrigins == "" {
		allowedOrigins = "*"
	}

	config := cors.DefaultConfig()
	config.AllowOrigins = strings.Split(allowedOrigins, ",")
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}

	return cors.New(config)
}
