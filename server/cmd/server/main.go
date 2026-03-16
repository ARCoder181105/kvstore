package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ARCoder181105/kvstore/internal/server"
	"github.com/ARCoder181105/kvstore/internal/store"
)

func main() {
	// create the store
	s := store.New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go s.StartEviction(ctx)

	// create the server
	srv := server.New(":6379", s)

	// start the server
	err := srv.Start()
	if err != nil {
		fmt.Println("failed to start server:", err)
		os.Exit(1)
	}

	fmt.Println("server listening on :6379")

	// block until Ctrl+C
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit // blocks here, waiting for signal

	fmt.Println("shutting down...")
	srv.Stop()
	fmt.Println("bye")
}
