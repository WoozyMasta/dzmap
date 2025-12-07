// Package config handles configuration loading and shared data structures.
package config

import (
	"os"

	"github.com/woozymasta/dzmap/internal/geo"

	"gopkg.in/yaml.v3"
)

// Config represents the root configuration file structure.
type Config struct {
	Attribution string `yaml:"attribution,omitempty" json:"attribution,omitempty"`
	Maps        []Map  `yaml:"maps" json:"maps"`
	ZoomLimit   int    `yaml:"zoom,omitempty"`
}

// Map represents a single game map configuration.
type Map struct {
	Index *int `yaml:"index,omitempty" json:"index,omitempty"`

	// defining GeoJSON directly in config.yaml
	LocationsInline *geo.GeoJSONFeatureCollection `yaml:"locations_geojson,omitempty" json:"-"`

	Name              string   `yaml:"name" json:"name"`
	Topographic       string   `yaml:"topographic" json:"-"`
	Satellite         string   `yaml:"satellite" json:"-"`
	LocationsURL      string   `yaml:"locations,omitempty" json:"-"`
	Attribution       string   `yaml:"attribution,omitempty" json:"attribution,omitempty"`
	Aliases           []string `yaml:"aliases,omitempty" json:"-"`
	ID                uint64   `yaml:"id" json:"id"` // Steam Workshop or App ID
	ZoomLimit         int      `yaml:"zoom,omitempty" json:"zoom"`
	Size              int      `yaml:"size,omitempty" json:"size"`
	TileSize          int      `yaml:"tile_size,omitempty" json:"-"` // only when processing single image
	LocationsIzurvive bool     `yaml:"locations_izurvive,omitempty" json:"-"`
	NoTopographic     bool     `yaml:"-" json:"no_topographic,omitempty"`
	NoSatellite       bool     `yaml:"-" json:"no_satellite,omitempty"`
}

// Load reads and parses the YAML configuration file from the specified path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
