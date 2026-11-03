package inventory

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Store serializes modifications and writes a complete JSON snapshot atomically.
// A shared data file survives switching between Blue and Green releases.
type Store struct {
	mu    sync.Mutex
	path  string
	items []Item
}

func NewStore(path string) (*Store, error) {
	store := &Store{path: path}
	contents, err := os.ReadFile(path)
	if err == nil {
		if err := json.Unmarshal(contents, &store.items); err != nil {
			return nil, fmt.Errorf("parse inventory: %w", err)
		}
		return store, nil
	}
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read inventory: %w", err)
	}
	store.items = DemoItems()
	if err := store.save(store.items); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) List() []Item {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := append([]Item(nil), s.items...)
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

func (s *Store) Summary() Summary {
	s.mu.Lock()
	defer s.mu.Unlock()
	return Summarize(s.items)
}

func (s *Store) Add(input ItemInput) (Item, error) {
	cleaned, err := input.Normalize()
	if err != nil {
		return Item{}, err
	}
	bytes := make([]byte, 5)
	if _, err := rand.Read(bytes); err != nil {
		return Item{}, err
	}
	item := Item{
		ID:   "SP-" + hex.EncodeToString(bytes),
		Name: cleaned.Name, Category: cleaned.Category, Quantity: cleaned.Quantity,
		MinStock: cleaned.MinStock, Unit: cleaned.Unit, UpdatedAt: time.Now().UTC(),
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	next := append(append([]Item(nil), s.items...), item)
	if err := s.save(next); err != nil {
		return Item{}, err
	}
	s.items = next
	return item, nil
}

func (s *Store) Update(id string, input UpdateInput) (Item, error) {
	if input.Quantity == nil && input.MinStock == nil {
		return Item{}, ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	next := append([]Item(nil), s.items...)
	for i := range next {
		if next[i].ID != id {
			continue
		}
		if input.Quantity != nil {
			next[i].Quantity = *input.Quantity
		}
		if input.MinStock != nil {
			next[i].MinStock = *input.MinStock
		}
		if next[i].Quantity < 0 || next[i].Quantity > 1000000 ||
			next[i].MinStock < 0 || next[i].MinStock > 1000000 {
			return Item{}, ErrInvalid
		}
		next[i].UpdatedAt = time.Now().UTC()
		if err := s.save(next); err != nil {
			return Item{}, err
		}
		s.items = next
		return next[i], nil
	}
	return Item{}, ErrNotFound
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := make([]Item, 0, len(s.items))
	found := false
	for _, item := range s.items {
		if item.ID == id {
			found = true
			continue
		}
		next = append(next, item)
	}
	if !found {
		return ErrNotFound
	}
	if err := s.save(next); err != nil {
		return err
	}
	s.items = next
	return nil
}

func (s *Store) save(items []Item) error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create data directory: %w", err)
	}
	encoded, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".stockpilot-*.json")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err := tmp.Write(encoded); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, s.path); err != nil {
		return err
	}
	return nil
}
