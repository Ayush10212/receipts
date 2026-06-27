package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Backend interface {
	Get(key string, out any) (bool, error)
	Put(key string, val any) error
}

// DiskBackend stores JSON-encoded entries under a directory, one file per key.
type DiskBackend struct {
	Dir string
}

func NewDiskBackend(dir string) (*DiskBackend, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &DiskBackend{Dir: dir}, nil
}

// Key builds a cache key from dist name, version, and dotted symbol path.
func Key(dist, version, symbol string) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%s\x00%s", dist, version, symbol)))
	return hex.EncodeToString(h[:])
}

func (d *DiskBackend) path(key string) string {
	return filepath.Join(d.Dir, key+".json")
}

func (d *DiskBackend) Get(key string, out any) (bool, error) {
	data, err := os.ReadFile(d.path(key))
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, json.Unmarshal(data, out)
}

func (d *DiskBackend) Put(key string, val any) error {
	data, err := json.Marshal(val)
	if err != nil {
		return err
	}
	return os.WriteFile(d.path(key), data, 0o644)
}
