package server

import (
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"sync"
)

type HashFS struct {
	httpFs http.FileSystem
	serv   http.Handler
	hashes sync.Map
}

func NewHashFS(fsys fs.FS) (*HashFS, error) {
	httpFs := http.FS(fsys)
	h := &HashFS{
		httpFs: httpFs,
		serv:   http.FileServer(httpFs),
	}

	// Walk the filesystem and compute hashes
	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		// Compute hash
		slog.Debug("computing static asset hash", "path", path)
		f, err := fsys.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		hash := sha256.New()
		if _, err := io.Copy(hash, f); err != nil {
			return err
		}
		hashStr := fmt.Sprintf("%x", hash.Sum(nil))
		slog.Debug("computed static asset hash", "path", path, "hash", hashStr)
		h.hashes.Store(path, hashStr)
		return nil
	})

	return h, err
}

func (h *HashFS) GetHash(path string) string {
	if val, ok := h.hashes.Load(path); ok {
		return val.(string)
	}
	return ""
}

func (h *HashFS) FormatWithHash(path string) string {
	hash := h.GetHash(path)
	if hash != "" {
		return fmt.Sprintf("%s?hash=%s", path, hash)
	}
	return path
}

func (h *HashFS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	hash := r.URL.Query().Get("hash")
	if hash != "" && hash == h.GetHash(r.URL.Path) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	}
	h.serv.ServeHTTP(w, r)
}
