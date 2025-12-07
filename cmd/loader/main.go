package main

import (
	"crypto/tls"
	"net/http"
	"os"
	"time"

	"github.com/woozymasta/dzmap/internal/config"
	"github.com/woozymasta/dzmap/internal/logger"
	"github.com/woozymasta/dzmap/internal/processor"

	"github.com/jessevdk/go-flags"
	"github.com/rs/zerolog/log"
)

type Options struct {
	Logger logger.Logger `group:"Logger options"`

	ConfigFile  string   `short:"c" long:"config"       env:"CONFIG_FILE"  description:"Path to configuration file" default:"config.yaml"`
	Limit       []string `short:"l" long:"limit"        env:"LIMIT_NAMES"  description:"Limit processing to specific map names"`
	Concurrency int      `short:"p" long:"concurrency"  env:"CONCURRENCY"  description:"Concurrency" default:"50"`
	ZoomLimit   int      `short:"z" long:"zoom-limit"   env:"ZOOM_LIMIT"   description:"Tiles zoom limit" default:"6"`
	TilesOnly   bool     `short:"t" long:"tiles-only"   description:"Download tiles only"`
	GeoJSONOnly bool     `short:"g" long:"geojson-only" description:"Generate GeoJSON only"`
	Force       bool     `short:"f" long:"force"        description:"Force overwrite of existing files"`
	FastCheck   bool     `short:"F" long:"fast-check"   description:"Skip processing if cache exist"`
}

func main() {
	var opts Options
	parser := flags.NewParser(&opts, flags.Default)
	if _, err := parser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		}
		os.Exit(1)
	}

	opts.Logger.Setup()

	cfg, err := config.Load(opts.ConfigFile)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	processTiles := true
	processGeo := true
	if opts.TilesOnly && !opts.GeoJSONOnly {
		processGeo = false
	} else if opts.GeoJSONOnly && !opts.TilesOnly {
		processTiles = false
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSNextProto:        make(map[string]func(string, *tls.Conn) http.RoundTripper),
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
		},
		Timeout: 15 * time.Second,
	}

	if opts.Concurrency <= 0 {
		opts.Concurrency = 50
	}

	if cfg.ZoomLimit <= 0 {
		if opts.ZoomLimit <= 0 {
			cfg.ZoomLimit = 6
		} else {
			cfg.ZoomLimit = opts.ZoomLimit
		}
	}

	// Filter maps if limit is set
	mapsToProcess := cfg.Maps
	if len(opts.Limit) > 0 {
		mapsToProcess = make([]config.Map, 0)
		availableMaps := make(map[string]config.Map)
		for _, m := range cfg.Maps {
			availableMaps[m.Name] = m
		}

		seen := make(map[string]bool)

		for _, limitName := range opts.Limit {
			if seen[limitName] {
				continue
			}
			seen[limitName] = true

			if m, ok := availableMaps[limitName]; ok {
				mapsToProcess = append(mapsToProcess, m)
			} else {
				log.Error().
					Str("name", limitName).
					Msg("Map specified in --limit not found in configuration")
			}
		}
	}

	log.Info().
		Int("maps_total", len(cfg.Maps)).
		Int("maps_queued", len(mapsToProcess)).
		Bool("fast_check", opts.FastCheck).
		Msg("Starting loader")

	for _, world := range mapsToProcess {
		hasLocations := world.LocationsURL != "" || world.LocationsInline != nil

		if hasLocations && processGeo {
			if err := processor.ProcessLocations(client, world, opts.Force); err != nil {
				log.Error().Err(err).Str("map", world.Name).Msg("Failed to process locations")
			}
		}

		if !processTiles {
			continue
		}

		processor.ProcessTiles(
			client,
			world,
			opts.Concurrency,
			cfg.ZoomLimit,
			opts.Force,
			opts.FastCheck)
	}

	log.Info().Msg("Loader finished successfully")
}
