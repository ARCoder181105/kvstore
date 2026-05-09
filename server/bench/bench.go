package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ARCoder181105/kvstore/internal/protocol"
)

func main() {
	conns := flag.Int("connections", 50, "Number of concurrent connections")
	duration := flag.Duration("duration", 10*time.Second, "Duration of the benchmark")
	host := flag.String("host", "localhost:6379", "TCP server address")
	flag.Parse()

	fmt.Printf("Starting benchmark: %d connections for %v on %s\n", *conns, *duration, *host)

	var opsCount atomic.Uint64

	startTime := time.Now()
	endTime := startTime.Add(*duration)

	var wg sync.WaitGroup

	for i := 0; i < *conns; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			conn, err := net.Dial("tcp", *host)
			if err != nil {
				fmt.Printf("Worker %d failed to connect: %v\n", workerID, err)
				return
			}
			defer conn.Close()

			val := []byte("benchmark_value")

			for time.Now().Before(endTime) {
				// Randomize key so we stress different parts of the map
				key := fmt.Sprintf("bench:%d", rand.Intn(10000))

				// SET
				if err := protocol.WriteCommand(conn, &protocol.Command{
					ID:    protocol.CmdSet,
					Key:   key,
					Value: val,
				}); err != nil {
					break // connection broken — exit this worker
				}
				if _, err := protocol.ReadResponse(conn); err != nil {
					break
				}
				opsCount.Add(1)

				// GET
				if err := protocol.WriteCommand(conn, &protocol.Command{
					ID:  protocol.CmdGet,
					Key: key,
				}); err != nil {
					break
				}
				if _, err := protocol.ReadResponse(conn); err != nil {
					break
				}
				opsCount.Add(1)
			}
		}(i)
	}

	wg.Wait()

	actualDuration := time.Since(startTime)
	totalOps := opsCount.Load()
	opsPerSec := float64(totalOps) / actualDuration.Seconds()

	fmt.Println("=====================================")
	fmt.Printf("Total Operations : %d\n", totalOps)
	fmt.Printf("Time Elapsed     : %.2f seconds\n", actualDuration.Seconds())
	fmt.Printf("Throughput       : %.0f ops/sec\n", opsPerSec)
	fmt.Println("=====================================")
}
