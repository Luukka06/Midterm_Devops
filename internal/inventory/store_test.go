package inventory

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func testStore(t *testing.T) (*Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "nested", "items.json")
	store, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	return store, path
}

func TestStoreStartsWithDemoItems(t *testing.T) {
	store, path := testStore(t)
	if got := len(store.List()); got != 7 {
		t.Fatalf("got %d demo items", got)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected persisted data: %v", err)
	}
}

func TestStoreAddAndReload(t *testing.T) {
	store, path := testStore(t)
	added, err := store.Add(ItemInput{Name: "USB hub", Quantity: 7, MinStock: 2})
	if err != nil {
		t.Fatal(err)
	}
	if added.ID == "" || added.Unit != "pcs" {
		t.Fatalf("unexpected added item: %+v", added)
	}
	reopened, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(reopened.List()); got != 8 {
		t.Fatalf("persistence failed: %d", got)
	}
}

func TestStoreUpdate(t *testing.T) {
	store, _ := testStore(t)
	quantity, minimum := 0, 9
	updated, err := store.Update("SP-101", UpdateInput{Quantity: &quantity, MinStock: &minimum})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Quantity != 0 || updated.MinStock != 9 || Status(updated) != "out" {
		t.Fatalf("unexpected update: %+v", updated)
	}
}

func TestStoreRejectsNegativeUpdate(t *testing.T) {
	store, _ := testStore(t)
	negative := -1
	if _, err := store.Update("SP-101", UpdateInput{Quantity: &negative}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("want ErrInvalid, got %v", err)
	}
}

func TestStoreDelete(t *testing.T) {
	store, _ := testStore(t)
	if err := store.Delete("SP-101"); err != nil {
		t.Fatal(err)
	}
	if len(store.List()) != 6 {
		t.Fatal("delete did not remove item")
	}
}

func TestStoreMissingItem(t *testing.T) {
	store, _ := testStore(t)
	count := 8
	if _, err := store.Update("missing", UpdateInput{Quantity: &count}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if err := store.Delete("missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestStoreConcurrentUpdates(t *testing.T) {
	store, _ := testStore(t)
	var wg sync.WaitGroup
	for n := 1; n <= 12; n++ {
		wg.Add(1)
		go func(value int) {
			defer wg.Done()
			if _, err := store.Update("SP-101", UpdateInput{Quantity: &value}); err != nil {
				t.Errorf("concurrent update: %v", err)
			}
		}(n)
	}
	wg.Wait()
	found := false
	for _, item := range store.List() {
		if item.ID == "SP-101" {
			found = item.Quantity >= 1 && item.Quantity <= 12
		}
	}
	if !found {
		t.Fatal("unexpected final quantity")
	}
}

func TestStoreRejectsCorruptedJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte("not JSON"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewStore(path); err == nil {
		t.Fatal("expected error for corrupted data")
	}
}
