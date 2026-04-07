package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ARCoder181105/kvstore/internal/api"
	aof "github.com/ARCoder181105/kvstore/internal/persistence"
	"github.com/ARCoder181105/kvstore/internal/raft"
	"github.com/ARCoder181105/kvstore/internal/server"
	"github.com/ARCoder181105/kvstore/internal/store"
)

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func main() {
	nodeID := getEnv("NODE_ID", "node1")
	httpAddr := getEnv("HTTP_ADDR", ":8080")
	tcpAddr := getEnv("TCP_ADDR", ":6379")
	dataDir := getEnv("DATA_DIR", "./data")
	peersStr := getEnv("PEERS", "node1=http://localhost:8080")

	peers := make(map[raft.NodeID]string)
	for _, p := range strings.Split(peersStr, ",") {
		parts := strings.Split(p, "=")
		if len(parts) == 2 {
			pID := raft.NodeID(parts[0])
			if pID != raft.NodeID(nodeID) {
				peers[pID] = parts[1]
			}
		}
	}

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		fmt.Println("failed to create data directory:", err)
		os.Exit(1)
	}

	s := store.New()

	// Restore all data BEFORE starting the eviction goroutine.
	// Previously, StartEviction launched first and could race-evict keys
	// with short remaining TTLs that were being loaded from disk.

	// 1. Load snapshot (absolute ExpiresAt — no time math required)
	if err := aof.Load(dataDir+"/snapshot.db", s); err != nil {
		fmt.Println("snapshot load error:", err)
		os.Exit(1)
	}

	// 2. Replay AOF on top of snapshot (also uses absolute ExpiresAt now)
	if err := aof.Replay(dataDir+"/aof.log", s); err != nil {
		fmt.Println("AOF replay error:", err)
		os.Exit(1)
	}

	// 3. NOW it is safe to start background eviction
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go s.StartEviction(ctx)

	// 4. Start AOF writer
	aofWriter, err := aof.NewAOFWriter(dataDir + "/aof.log")
	if err != nil {
		fmt.Println("failed to create AOF writer:", err)
		os.Exit(1)
	}
	go aofWriter.Start(ctx)

	// 5. Start TCP server
	srv := server.New(tcpAddr, s, aofWriter)
	if err := srv.Start(); err != nil {
		fmt.Println("failed to start server:", err)
		os.Exit(1)
	}
	fmt.Printf("kvstore listening on %s\n", tcpAddr)

	// 6. Initialize Raft
	raftNode := raft.New(raft.NodeID(nodeID), peers, s, aofWriter)

	apiSrv := api.New(s, raftNode)
	if err := apiSrv.Start(httpAddr); err != nil {
		fmt.Println("failed to start HTTP server:", err)
		os.Exit(1)
	}
	fmt.Printf("HTTP API listening on %s\n", httpAddr)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("shutting down...")
	srv.Stop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	apiSrv.Stop(shutdownCtx)

	cancel() // stops eviction + AOF goroutines

	// Save final snapshot on clean shutdown
	if err := aof.Save(s, dataDir+"/snapshot.db"); err != nil {
		fmt.Println("snapshot save error:", err)
	}

	fmt.Println("bye")
}
