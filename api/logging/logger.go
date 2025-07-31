package logging

import (
	"io"
	"time"

	"github.com/rs/zerolog"
)

const (
	FieldChain  = "chain"
	FieldBlock  = "block_number"
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
