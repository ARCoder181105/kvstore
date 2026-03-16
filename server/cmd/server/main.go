package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	aof "github.com/ARCoder181105/kvstore/internal/persistence"
	"github.com/ARCoder181105/kvstore/internal/server"
	"github.com/ARCoder181105/kvstore/internal/store"
)

func main() {
	if err := os.MkdirAll("./data", 0755); err != nil {
		fmt.Println("failed to create data directory:", err)
		os.Exit(1)
	}
	// create the store
	s := store.New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go s.StartEviction(ctx)

	// load snapshot first
	if err := aof.Load("./data/snapshot.db", s); err != nil {
		fmt.Println("snapshot load error:", err)
		os.Exit(1)
	}

	// replay AOF on top
	if err := aof.Replay("./data/aof.log", s); err != nil {
		fmt.Println("AOF replay error:", err)
		os.Exit(1)
	}

	// start AOF writer
	aofWriter, err := aof.NewAOFWriter("./data/aof.log")
	if err != nil {
		fmt.Println("failed to create AOF writer:", err)
		os.Exit(1)
	}
	go aofWriter.Start(ctx)

	// create the server
	srv := server.New(":6379", s, aofWriter)

	// start the server
	err = srv.Start()
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

	// save snapshot on clean shutdown
	if err := aof.Save(s, "./data/snapshot.db"); err != nil {
		fmt.Println("snapshot save error:", err)
	}

	fmt.Println("bye")
}
