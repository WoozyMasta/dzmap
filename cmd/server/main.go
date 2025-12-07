package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/woozymasta/dzmap/internal/config"
	"github.com/woozymasta/dzmap/internal/logger"
	"github.com/woozymasta/dzmap/internal/server"

	"github.com/jessevdk/go-flags"
	"github.com/rs/zerolog/log"
)

type Options struct {
	Logger logger.Logger `group:"Logger options"`

	ConfigFile string `short:"c" long:"config"     env:"CONFIG_FILE"    description:"Path to configuration file" default:"config.yaml"`
	Addr       string `short:"a" long:"addr"       env:"LISTEN_ADDRESS" description:"Address to listen on"       default:"0.0.0.0"`
	Port       int    `short:"p" long:"port"       env:"LISTEN_PORT"    description:"Port to listen on"          default:"8080"`
	ZoomLimit  int    `short:"z" long:"zoom-limit" env:"ZOOM_LIMIT"     description:"Tiles zoom limit"           default:"6"`
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

	// Setup Logging
	opts.Logger.Setup()

	// Load Config
	cfg, err := config.Load(opts.ConfigFile)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	if cfg.ZoomLimit <= 0 {
		if opts.ZoomLimit <= 0 {
			cfg.ZoomLimit = 6
		} else {
			cfg.ZoomLimit = opts.ZoomLimit
		}
	}

	srvCtx := server.NewServerContext(cfg)

	// Routes
	mux := http.NewServeMux()
	mux.HandleFunc("/api/maps", srvCtx.HandleMapsList)
	mux.HandleFunc("/favicon.ico", srvCtx.HandleFavicon)
	mux.HandleFunc("/maps/", srvCtx.HandleTileOrLoc)
	mux.HandleFunc("/", srvCtx.HandleIndex)

	handler := server.RequestLogger(mux)

	listenAddr := fmt.Sprintf("%s:%d", opts.Addr, opts.Port)
	log.Info().
		Str("addr", listenAddr).
		Int("maps_loaded", len(cfg.Maps)).
		Int("default_zoom", cfg.ZoomLimit).
		Msg("Web server started")

	if err := http.ListenAndServe(listenAddr, handler); err != nil {
		log.Fatal().Err(err).Msg("Server failed")
	}
}
