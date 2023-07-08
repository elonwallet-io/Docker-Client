package main

import (
	"deployer/server"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"os/signal"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	if err := run(); err != nil {
		log.Fatal().Caller().Err(err).Msg("failed to start")
	}
}

func run() error {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	s, err := server.New()
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	go func() {
		if err := s.Run(); err != nil {
			log.Fatal().Caller().Err(err).Msg("failed to start")
		}
	}()

	<-stop

	return s.Stop()
}
