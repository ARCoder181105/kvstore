package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ARCoder181105/kvstore/internal/store"
)

func TestHandleHealth(t *testing.T) {
	s := store.New()
	apiSrv := New(s)

	req := httptest.NewRequest("GET", "/api/health", nil)
	rec := httptest.NewRecorder()
	apiSrv.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", rec.Code)
	}
	var body map[string]string
	json.NewDecoder(rec.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Fatalf("expected status ok got %s", body["status"])
	}
}

func TestHandleGetKeyFound(t *testing.T) {
	s := store.New()
	s.Set("foo", []byte("bar"), 0)
	apiSrv := New(s)

	req := httptest.NewRequest("GET", "/api/keys/foo", nil)
	rec := httptest.NewRecorder()
	apiSrv.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", rec.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["key"] != "foo" {
		t.Fatalf("expected key foo got %s", body["key"])
	}
	if body["value"] != "bar" {
		t.Fatalf("expected value bar got %s", body["value"])
	}
}

func TestHandleGetKeyNotFound(t *testing.T) {
	s := store.New()
	apiSrv := New(s)

	req := httptest.NewRequest("GET", "/api/keys/foo", nil)
	rec := httptest.NewRecorder()
	apiSrv.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", rec.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["error"] == "" {
		t.Fatalf("expected error message, got empty string")
	}
}

func TestHandleSetKey(t *testing.T) {
	s := store.New()
	apiSrv := New(s)

	// key goes in the URL, only value (and optional ttl) in the body
	payload := `{"value":"bar"}`
	req := httptest.NewRequest("POST", "/api/keys/foo",
		bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	apiSrv.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", rec.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["key"] != "foo" {
		t.Fatalf("expected key foo got %v", body["key"])
	}
	if body["value"] != "bar" {
		t.Fatalf("expected value bar got %v", body["value"])
	}
}

func TestHandleSetKeyMissingValue(t *testing.T) {
	s := store.New()
	apiSrv := New(s)

	// empty value field → handler returns 400
	payload := `{"value":""}`
	req := httptest.NewRequest("POST", "/api/keys/foo",
		bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	apiSrv.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", rec.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["error"] == "" {
		t.Fatalf("expected error message, got empty string")
	}
}

func TestHandleDeleteKey(t *testing.T) {
	s := store.New()
	s.Set("foo", []byte("bar"), 0)
	apiSrv := New(s)

	req := httptest.NewRequest("DELETE", "/api/keys/foo", nil)
	rec := httptest.NewRecorder()
	apiSrv.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", rec.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	// handler returns {"key":"foo","deleted":true}
	if body["deleted"] != true {
		t.Fatalf("expected deleted:true got %v", body["deleted"])
	}
}

func TestHandleDeleteKeyNotFound(t *testing.T) {
	s := store.New()
	apiSrv := New(s)

	req := httptest.NewRequest("DELETE", "/api/keys/foo", nil)
	rec := httptest.NewRecorder()
	apiSrv.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", rec.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["error"] == "" {
		t.Fatalf("expected error message, got empty string")
	}
}

func TestHandleListKeys(t *testing.T) {
	s := store.New()
	s.Set("foo", []byte("bar"), 0)
	s.Set("baz", []byte("qux"), 0)
	apiSrv := New(s)

	req := httptest.NewRequest("GET", "/api/keys", nil)
	rec := httptest.NewRecorder()
	apiSrv.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", rec.Code)
	}
	// handler returns {"keys":[{"key":"...","value":"...","ttl":...}],"count":2}
	var body struct {
		Keys  []map[string]any `json:"keys"`
		Count int              `json:"count"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body.Count != 2 {
		t.Fatalf("expected count 2 got %d", body.Count)
	}
	keySet := make(map[string]struct{})
	for _, entry := range body.Keys {
		if k, ok := entry["key"].(string); ok {
			keySet[k] = struct{}{}
		}
	}
	if _, ok := keySet["foo"]; !ok {
		t.Fatalf("expected key foo in list")
	}
	if _, ok := keySet["baz"]; !ok {
		t.Fatalf("expected key baz in list")
	}
}
