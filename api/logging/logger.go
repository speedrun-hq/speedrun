package logging

import (
	"io"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

const (
	FieldChain  = "chain"
	FieldBlock  = "block_number"
	FieldIntent = "intent_id"
	FieldModule = "module"
)

func New(writer io.Writer, level zerolog.Level, jsonOutput bool) zerolog.Logger {
	if !jsonOutput {
		writer = zerolog.ConsoleWriter{
			Out:        writer,
			TimeFormat: time.RFC3339,
		}
	}

	return zerolog.New(writer).Level(level).With().Timestamp().Caller().Logger()
}

func NewTesting(t *testing.T) zerolog.Logger {
	return New(zerolog.NewTestWriter(t), zerolog.DebugLevel, true)
}
