package httpapi

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type spaHandler struct {
	directory string
	indexFile string
	files     http.Handler
}

func newSPAHandler(directory string) (http.Handler, error) {
	if strings.TrimSpace(directory) == "" {
		return nil, errors.New("static directory is required")
	}
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return nil, err
	}
	indexFile := filepath.Join(absolute, "index.html")
	info, err := os.Stat(indexFile)
	if err != nil {
		return nil, fmt.Errorf("stat SPA entrypoint: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("SPA entrypoint must be a regular file")
	}
	return &spaHandler{directory: absolute, indexFile: indexFile, files: http.FileServer(http.Dir(absolute))}, nil
}

func (h *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	cleanPath := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
	if cleanPath == "." {
		cleanPath = "index.html"
	}
	if cleanPath == ".." || strings.HasPrefix(cleanPath, ".."+string(filepath.Separator)) {
		http.NotFound(w, r)
		return
	}
	filePath := filepath.Join(h.directory, cleanPath)
	if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
		h.files.ServeHTTP(w, r)
		return
	} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if filepath.Ext(cleanPath) != "" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, h.indexFile)
}
