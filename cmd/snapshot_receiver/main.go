package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

const maxSnapshotBytes = 64 << 20

type snapshotMetadata struct {
	SchemaVersion int       `json:"schema_version"`
	SnapshotID    string    `json:"snapshot_id"`
	CapturedAt    time.Time `json:"captured_at"`
}

type storedSnapshot struct {
	metadata   snapshotMetadata
	payload    []byte
	receivedAt time.Time
}

type snapshotHandler struct {
	latest atomic.Pointer[storedSnapshot]
}

func main() {
	address := strings.TrimSpace(os.Getenv("SNAPSHOT_LISTEN_ADDR"))
	if address == "" {
		address = ":8081"
	}

	handler := &snapshotHandler{}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", handler.health)
	mux.HandleFunc("/internal/v1/snapshots", handler.receive)
	mux.HandleFunc("/internal/v1/snapshots/latest", handler.latestSnapshot)

	server := &http.Server{
		Addr:              address,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("snapshot receiver shutdown failed", slog.Any("error", err))
		}
	}()

	slog.Info("snapshot receiver listening", slog.String("address", address))
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("snapshot receiver stopped", slog.Any("error", err))
		os.Exit(1)
	}
}

func (h *snapshotHandler) health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, "ok")
}

func (h *snapshotHandler) receive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxSnapshotBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "snapshot body is too large or unreadable", http.StatusRequestEntityTooLarge)
		return
	}
	var metadata snapshotMetadata
	if err := json.Unmarshal(body, &metadata); err != nil {
		http.Error(w, "invalid snapshot JSON", http.StatusBadRequest)
		return
	}
	if metadata.SchemaVersion != 1 || metadata.SnapshotID == "" || metadata.CapturedAt.IsZero() {
		http.Error(w, "snapshot metadata is incomplete or unsupported", http.StatusBadRequest)
		return
	}

	entry := &storedSnapshot{
		metadata:   metadata,
		payload:    body,
		receivedAt: time.Now().UTC(),
	}
	for {
		previous := h.latest.Load()
		if previous != nil && !metadata.CapturedAt.After(previous.metadata.CapturedAt) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if h.latest.CompareAndSwap(previous, entry) {
			break
		}
	}

	slog.Info(
		"received market snapshot",
		slog.String("snapshot_id", metadata.SnapshotID),
		slog.Int("bytes", len(body)),
		slog.Time("captured_at", metadata.CapturedAt),
		slog.Time("received_at", entry.receivedAt),
	)
	w.WriteHeader(http.StatusNoContent)
}

func (h *snapshotHandler) latestSnapshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	entry := h.latest.Load()
	if entry == nil {
		http.Error(w, "no snapshot received yet", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Snapshot-ID", entry.metadata.SnapshotID)
	w.Header().Set("X-Snapshot-Received-At", entry.receivedAt.Format(time.RFC3339Nano))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(entry.payload)
}
