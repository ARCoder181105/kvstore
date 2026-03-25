package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

type keyEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	TTL   int64  `json:"ttl"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func (s *APIServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *APIServer) handleStats(w http.ResponseWriter, r *http.Request) {
	elapsed := time.Since(s.startTime).Round(time.Second).String()

	writeJSON(w, http.StatusOK, map[string]any{
		"total_keys":       s.store.Count(),
		"uptime":           elapsed,
		"memory_bytes":     s.store.MemoryUsage(),
		"ttl_keys":         s.store.TTLKeyCount(),
		"connected_clients": s.store.SubscriberCount(),
	})
}

func (s *APIServer) handleGetKey(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")

	val, ok := s.store.Get(key)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "key not found",
			"code":  "NOT_FOUND",
		})
		return
	}

	ttl := s.store.TTL(key)

	var ttlSeconds int64
	switch {
	case ttl == -1:
		ttlSeconds = -1 // no expiry
	case ttl == -2:
		ttlSeconds = -1 // edge case — treat as no expiry
	case ttl > 0:
		ttlSeconds = ttl / int64(time.Second)
		if ttlSeconds == 0 {
			ttlSeconds = 1 // less than 1s remaining, round up
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"key":   key,
		"value": string(val),
		"ttl":   ttlSeconds,
	})

}

func (s *APIServer) handleSetKey(w http.ResponseWriter, r *http.Request) {

	var body struct {
		Value string `json:"value"`
		TTL   int64  `json:"ttl"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
			"code":  "BAD_REQUEST",
		})
		return
	}

	if body.Value == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "value is required",
			"code":  "BAD_REQUEST",
		})
		return
	}

	key := chi.URLParam(r, "key")

	var ttlNs int64
	if body.TTL > 0 {
		ttlNs = body.TTL * int64(time.Second)
	}

	s.store.Set(key, []byte(body.Value), ttlNs)

	ttlResponse := body.TTL
	if ttlResponse == 0 {
		ttlResponse = -1 // no expiry
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"key":   key,
		"value": body.Value,
		"ttl":   ttlResponse,
	})

}

func (s *APIServer) handleDeleteKey(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")

	ok := s.store.Delete(key)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "key not found",
			"code":  "NOT_FOUND",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"key":     key,
		"deleted": true,
	})

}

func (s *APIServer) handleListKeys(w http.ResponseWriter, r *http.Request) {

	pattern := r.URL.Query().Get("pattern")
	if pattern == "" {
		pattern = "*"
	}

	keys := s.store.Keys(pattern)

	entries := make([]keyEntry, 0)

	for _, k := range keys {
		value, ok := s.store.Get(k)
		if !ok {
			continue
		}
		ttl := s.store.TTL(k)
		var ttlSeconds int64
		switch {
		case ttl == -1:
			ttlSeconds = -1
		case ttl == -2:
			ttlSeconds = -1
		case ttl > 0:
			ttlSeconds = ttl / int64(time.Second)
			if ttlSeconds == 0 {
				ttlSeconds = 1
			}
		}
		entries = append(entries, keyEntry{
			Key:   k,
			Value: string(value),
			TTL:   ttlSeconds,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"keys":  entries,
		"count": len(entries),
	})
}

func (s *APIServer) handleExpireKey(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")

	var body struct {
		Seconds int64 `json:"seconds"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
			"code":  "BAD_REQUEST",
		})
		return
	}

	if body.Seconds <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "seconds must be greater than 0",
			"code":  "BAD_REQUEST",
		})
		return
	}

	ok := s.store.Expire(key, body.Seconds*int64(time.Second))
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "key not found",
			"code":  "NOT_FOUND",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"key": key,
		"ttl": body.Seconds,
	})
}

func (s *APIServer) handleGetTTL(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")

	ttl := s.store.TTL(key)

	var ttlSeconds int64
	switch {
	case ttl == -1:
		ttlSeconds = -1
	case ttl == -2:
		ttlSeconds = -2
	case ttl > 0:
		ttlSeconds = ttl / int64(time.Second)
		if ttlSeconds == 0 {
			ttlSeconds = 1
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"key": key,
		"ttl": ttlSeconds,
	})
}
