package main

import (
	"context"
	"db_trace/kucoin-ising-bot/internal/config"
	"db_trace/kucoin-ising-bot/internal/core"
	kucoin "db_trace/kucoin-ising-bot/internal/exchange"
	"log"
	"os/signal"
	"syscall"
)

func main() {
	cfg := config.Load()

	engine, err := core.NewEngine(cfg)
	if err != nil {
		log.Fatalf("create engine: %v", err)
	}
	defer engine.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go engine.Run(ctx)

	client := kucoin.NewClient(cfg, engine.Events())

	if err := client.Run(ctx); err != nil && err != context.Canceled {
		log.Fatalf("client run: %v", err)
	}
}
