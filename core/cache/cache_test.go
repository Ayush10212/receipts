package cache_test

import (
	"os"
	"testing"

	"github.com/Ayush10212/receipts/core/cache"
)

func TestDiskBackend_GetPut(t *testing.T) {
	dir, err := os.MkdirTemp("", "receipts-cache-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	b, err := cache.NewDiskBackend(dir)
	if err != nil {
		t.Fatal(err)
	}

	key := cache.Key("pandas", "2.1.0", "DataFrame.merge")

	// Miss on empty cache.
	var result map[string]any
	hit, err := b.Get(key, &result)
	if err != nil || hit {
		t.Fatalf("expected miss, got hit=%v err=%v", hit, err)
	}

	// Put a value.
	val := map[string]any{"verdict": "grounded", "symbol": "DataFrame.merge"}
	if err := b.Put(key, val); err != nil {
		t.Fatalf("Put: %v", err)
	}

	// Hit.
	var got map[string]any
	hit, err = b.Get(key, &got)
	if err != nil {
		t.Fatalf("Get after Put: %v", err)
	}
	if !hit {
		t.Fatal("expected cache hit after Put")
	}
	if got["verdict"] != "grounded" {
		t.Errorf("got verdict %v want grounded", got["verdict"])
	}
}

func TestDiskBackend_KeyIsDeterministic(t *testing.T) {
	k1 := cache.Key("pandas", "2.1.0", "DataFrame.merge")
	k2 := cache.Key("pandas", "2.1.0", "DataFrame.merge")
	if k1 != k2 {
		t.Errorf("keys differ: %q vs %q", k1, k2)
	}
	k3 := cache.Key("pandas", "2.1.0", "DataFrame.append")
	if k1 == k3 {
		t.Error("different symbols should produce different keys")
	}
}
