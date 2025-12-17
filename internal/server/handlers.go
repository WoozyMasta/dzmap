// Package server handles HTTP requests and middleware.
package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const etagCap = 64

// HandleMapsList serves the JSON configuration of available maps.
func (s *ServerContext) HandleMapsList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// Ignoring error as we cannot handle client disconnects
	_ = json.NewEncoder(w).Encode(s.Config.Maps)
}

// HandleFavicon serves the site favicon.
func (s *ServerContext) HandleFavicon(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/favicon.ico" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "image/x-icon")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = w.Write(s.Favicon)
}

// HandleIndex serves the main HTML application.
func (s *ServerContext) HandleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && strings.Contains(r.URL.Path, ".") {
		http.NotFound(w, r)
		return
	}

	etag := fmt.Sprintf(`"%x"`, len(s.IndexHTML))

	if match := r.Header.Get("If-None-Match"); match == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "public, no-cache")
	_, _ = w.Write(s.IndexHTML)
}

// HandleTileOrLoc serves static assets (tiles and GeoJSON) for specific maps.
func (s *ServerContext) HandleTileOrLoc(w http.ResponseWriter, r *http.Request) {
	// Path: /maps/{mapName}/...
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")

	if len(parts) < 3 {
		http.NotFound(w, r)
		return
	}

	requestedName := parts[1]
	realMapName, ok := s.MapNameResolver[requestedName]
	if !ok {
		http.NotFound(w, r)
		return
	}

	// GeoJSON
	if len(parts) == 3 && parts[2] == "locations.geojson" {
		path := filepath.Join("maps", realMapName, "locations.geojson")
		s.serveFile(w, r, path, "application/geo+json")
		return
	}

	// WebP Tile
	if len(parts) >= 6 {
		// parts: maps, mapName, layer, z, x, y.webp
		layer, z, x, y := parts[2], parts[3], parts[4], parts[5]

		// allow only known layers to prevent path probing
		if layer != "topographic" && layer != "satellite" {
			http.NotFound(w, r)
			return
		}

		tryServe := func(l string) bool {
			path := filepath.Join("maps", realMapName, l, z, x, y)
			return s.serveFile(w, r, path, "")
		}

		// try requested layer
		if tryServe(layer) {
			return
		}

		// fallback to the other layer
		alt := "satellite"
		if layer == "satellite" {
			alt = "topographic"
		}
		if tryServe(alt) {
			return
		}

		// cache transparent tile
		w.Header().Set("Content-Type", "image/webp")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		_, _ = w.Write(s.TransparentTile)
		return
	}

	http.NotFound(w, r)
}

// serveFile tries to serve a file from disk with ETag generation.
// It returns true if the file was found and served (or 304).
func (s *ServerContext) serveFile(w http.ResponseWriter, r *http.Request, path string, contentType string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	if info.IsDir() {
		return false
	}

	buf := make([]byte, 0, etagCap)
	buf = append(buf, '"')
	buf = strconv.AppendInt(buf, info.Size(), 16)
	buf = append(buf, '-')
	buf = strconv.AppendInt(buf, info.ModTime().UnixNano(), 16)
	buf = append(buf, '"')
	etag := string(buf)

	// check If-None-Match (client sent ETag)
	if match := r.Header.Get("If-None-Match"); match == etag {
		w.WriteHeader(http.StatusNotModified)
		return true
	}

	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "public, no-cache")

	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}

	http.ServeFile(w, r, path)
	return true
}
