package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ARCoder181105/kvstore/internal/api"
	aof "github.com/ARCoder181105/kvstore/internal/persistence"
	"github.com/ARCoder181105/kvstore/internal/server"
	"github.com/ARCoder181105/kvstore/internal/store"
)

func main() {
	if err := os.MkdirAll("./data", 0755); err != nil {
		fmt.Println("failed to create data directory:", err)
		os.Exit(1)
	}

	s := store.New()

	// Restore all data BEFORE starting the eviction goroutine.
	// Previously, StartEviction launched first and could race-evict keys
	// with short remaining TTLs that were being loaded from disk.

	// 1. Load snapshot (absolute ExpiresAt — no time math required)
	if err := aof.Load("./data/snapshot.db", s); err != nil {
		fmt.Println("snapshot load error:", err)
		os.Exit(1)
	}

	// 2. Replay AOF on top of snapshot (also uses absolute ExpiresAt now)
	if err := aof.Replay("./data/aof.log", s); err != nil {
		fmt.Println("AOF replay error:", err)
		os.Exit(1)
	}

	// 3. NOW it is safe to start background eviction
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go s.StartEviction(ctx)

	// 4. Start AOF writer
	aofWriter, err := aof.NewAOFWriter("./data/aof.log")
	if err != nil {
		fmt.Println("failed to create AOF writer:", err)
		os.Exit(1)
	}
	go aofWriter.Start(ctx)

	// 5. Start TCP server
	srv := server.New(":6379", s, aofWriter)
	if err := srv.Start(); err != nil {
		fmt.Println("failed to start server:", err)
		os.Exit(1)
	}
	fmt.Println("kvstore listening on :6379")

	apiSrv := api.New(s)
	if err := apiSrv.Start(":8080"); err != nil {
		fmt.Println("failed to start HTTP server:", err)
		os.Exit(1)
	}
	fmt.Println("HTTP API listening on :8080")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("shutting down...")
	srv.Stop()
	cancel() // stops eviction + AOF goroutines

	// Save final snapshot on clean shutdown
	if err := aof.Save(s, "./data/snapshot.db"); err != nil {
		fmt.Println("snapshot save error:", err)
	}

	fmt.Println("bye")
}
