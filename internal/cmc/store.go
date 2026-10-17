package cmc

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const DefaultSnapshotPath = "data/cmc_snapshot.json"

func NewStore(path string) (*Store, error) {
	store := &Store{path: path}
	empty := Snapshot{}
	store.cur.Store(&empty)

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return store, fmt.Errorf("read CMC snapshot %q: %w", path, err)
	}

	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return store, fmt.Errorf("decode CMC snapshot %q: %w", path, err)
	}
	if snapshot == nil {
		return store, fmt.Errorf("decode CMC snapshot %q: snapshot is null", path)
	}

	store.cur.Store(&snapshot)
	return store, nil
}

func (s *Store) publish(snapshot *Snapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	persistErr := s.persist(snapshot)
	s.cur.Store(snapshot)
	return persistErr
}

func (s *Store) persist(snapshot *Snapshot) error {
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("encode CMC snapshot: %w", err)
	}

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create CMC snapshot directory %q: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, ".cmc-snapshot-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary CMC snapshot: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temporary CMC snapshot: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync temporary CMC snapshot: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary CMC snapshot: %w", err)
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		return fmt.Errorf("replace CMC snapshot %q: %w", s.path, err)
	}

	return nil
}

func (s *Store) Get() *Snapshot { return s.cur.Load() }
