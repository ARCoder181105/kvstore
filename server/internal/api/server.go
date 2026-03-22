package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/ARCoder181105/kvstore/internal/store"
	"github.com/go-chi/chi/v5"
)

type APIServer struct {
	store     *store.Store
	router    chi.Router
	startTime time.Time
}

func (s *APIServer) setupRoutes() {
	r := chi.NewRouter()

	// mount handlers here — s.store is accessible
	r.Get("/api/health", s.handleHealth)
	r.Get("/api/stats", s.handleStats)

	r.Get("/api/keys", s.handleListKeys)
	r.Post("/api/keys/{key}", s.handleSetKey)
	r.Get("/api/keys/{key}", s.handleGetKey)
	r.Delete("/api/keys/{key}", s.handleDeleteKey)
	r.Put("/api/keys/{key}/expire", s.handleExpireKey)
	r.Get("/api/keys/{key}/ttl", s.handleGetTTL)
	r.Get("/ws/events", s.handleWebSocket)

	s.router = r
}

func New(store *store.Store) *APIServer {
	s := &APIServer{
		store:     store,
		router:    chi.NewRouter(),
		startTime: time.Now(),
	}
	s.setupRoutes()
	return s
}

func (s *APIServer) Start(addr string) error {
	httpServer := &http.Server{
		Addr:    addr,
		Handler: s.router,
	}
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("http server error: %v\n", err)
		}
	}()
	return nil
}
