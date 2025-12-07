package server

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/rs/zerolog/log"
	"github.com/woozymasta/dzmap/assets"
	"github.com/woozymasta/dzmap/internal/config"
)

// ServerContext holds dependencies for request handlers.
type ServerContext struct {
	Config          *config.Config
	MapNameResolver map[string]string
	IndexHTML       []byte
	Favicon         []byte
	TransparentTile []byte
}

// NewServerContext initializes the context and processes the map configuration.
// It filters out maps with missing assets and sets up the name resolver.
func NewServerContext(cfg *config.Config) *ServerContext {
	log.Info().Int("config_maps_count", len(cfg.Maps)).Msg("Initializing server context")

	resolver := make(map[string]string)
	validMaps := make([]config.Map, 0, len(cfg.Maps))

	// Normalize and Sort
	for i := range cfg.Maps {
		world := &cfg.Maps[i]

		if world.ZoomLimit <= 0 {
			world.ZoomLimit = cfg.ZoomLimit
		}
		if world.Attribution == "" {
			world.Attribution = cfg.Attribution
		}

		// Check for cache existence
		mapBaseDir := filepath.Join("maps", world.Name)

		// Check Topographic
		if world.Topographic == "" {
			world.NoTopographic = true
			log.Trace().
				Str("map", world.Name).
				Msg("Topographic layer skipped: no source in config")
		} else {
			topoDir := filepath.Join(mapBaseDir, "topographic")
			if _, err := os.Stat(topoDir); os.IsNotExist(err) {
				world.NoTopographic = true
				log.Trace().
					Str("map", world.Name).
					Str("path", topoDir).
					Msg("Topographic layer skipped: directory not found")
			} else {
				log.Trace().
					Str("map", world.Name).
					Msg("Topographic layer found")
			}
		}

		// Check Satellite
		if world.Satellite == "" {
			world.NoSatellite = true
			log.Trace().
				Str("map", world.Name).
				Msg("Satellite layer skipped: no source in config")
		} else {
			satDir := filepath.Join(mapBaseDir, "satellite")
			if _, err := os.Stat(satDir); os.IsNotExist(err) {
				world.NoSatellite = true
				log.Trace().
					Str("map", world.Name).
					Str("path", satDir).
					Msg("Satellite layer skipped: directory not found")
			} else {
				log.Trace().
					Str("map", world.Name).
					Msg("Satellite layer found")
			}
		}

		if world.NoTopographic && world.NoSatellite {
			log.Warn().
				Str("map", world.Name).
				Msg("Skipping map: no valid layers found (neither topographic nor satellite)")
			continue
		}

		// Setup Resolver
		resolver[world.Name] = world.Name
		for _, alias := range world.Aliases {
			resolver[alias] = world.Name
		}

		log.Debug().
			Str("map", world.Name).
			Bool("topo", !world.NoTopographic).
			Bool("sat", !world.NoSatellite).
			Msg("Map validated and added to context")

		validMaps = append(validMaps, *world)
	}

	cfg.Maps = validMaps

	sort.Slice(cfg.Maps, func(i, j int) bool {
		idxI, idxJ := 999999, 999999
		if cfg.Maps[i].Index != nil {
			idxI = *cfg.Maps[i].Index
		}
		if cfg.Maps[j].Index != nil {
			idxJ = *cfg.Maps[j].Index
		}
		if idxI != idxJ {
			return idxI < idxJ
		}

		return cfg.Maps[i].Name < cfg.Maps[j].Name
	})

	log.Info().
		Int("valid_maps_count", len(cfg.Maps)).
		Msg("Server context initialized successfully")

	return &ServerContext{
		Config:          cfg,
		IndexHTML:       assets.Index,
		Favicon:         assets.Favicon,
		TransparentTile: assets.TransparentTile,
		MapNameResolver: resolver,
	}
}
