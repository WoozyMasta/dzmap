// Package processor handles the downloading and processing of map data.
package processor

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/woozymasta/dzmap/internal/config"
	"github.com/woozymasta/dzmap/internal/geo"

	"github.com/rs/zerolog/log"
)

// ProcessLocations handles the logic for fetching and converting location data.
// It supports inline data from config, iZurvive, and Xam formats.
func ProcessLocations(client *http.Client, m config.Map, force bool) error {
	destDir := filepath.Join("maps", m.Name)
	destFile := filepath.Join(destDir, "locations.geojson")

	// Check if file exists
	if _, err := os.Stat(destFile); err == nil {
		if !force {
			log.Debug().Str("map", m.Name).Msg("Locations file exists, skipping")
			return nil
		}
	}

	var fc geo.GeoJSONFeatureCollection
	var err error

	// Inline Data Priority
	if m.LocationsInline != nil {
		log.Info().
			Str("map", m.Name).
			Msg("Using inline locations data from config")
		fc = *m.LocationsInline

	} else if m.LocationsURL != "" {
		// Download Data
		log.Info().
			Str("map", m.Name).
			Str("source", m.LocationsURL).
			Msg("Processing locations from URL")

		if m.LocationsIzurvive {
			fc, err = fetchIzurvive(client, m.LocationsURL)
		} else {
			// Default size fallback if missing
			size := m.Size
			if size == 0 {
				log.Warn().
					Str("map", m.Name).
					Msg("Map size not set, defaulting to 15360 for Xam calculation")
				size = 15360
			}
			fc, err = fetchXam(client, m.LocationsURL, size)
		}

	} else {
		return nil
	}

	if err != nil {
		return err
	}

	return saveGeoJSON(destDir, destFile, fc)
}

// saveGeoJSON marshals the feature collection and writes it to disk.
func saveGeoJSON(dir, path string, fc geo.GeoJSONFeatureCollection) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}

	// We care about write errors on close
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			log.Error().Err(closeErr).Str("path", path).Msg("Failed to close file")
		}
	}()

	return json.NewEncoder(f).Encode(fc)
}
