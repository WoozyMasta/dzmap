# DZMap

A self-hosted tile server and map data normalizer for DayZ.

## Motivation

This project was created to support coordinate visualization for [MetricZ]
and to utilize dynamic tile layers in Grafana (via PR [#114371]).

While community projects like [xam.nu] and [iZurvive] provide excellent
data, relying on them for external integrations presents several technical
challenges:

* **Inconsistency:** Map names often differ from engine names. Tile formats
  vary (WebP, PNG, JPG) with no standard.
* **No Stability:** There is no public API for versions. URL paths often
  contain version numbers, and old versions are deleted, breaking external
  links.
* **Coordinates:** [xam.nu] uses a custom coordinate system; [iZurvive] has
  instances of distorted tile scaling.
* **Data Formats:** Location data is stored in custom, proprietary formats
  rather than standards like GeoJSON.

**DZMap** resolves these issues by downloading, normalizing, and serving map
data via stable URLs with standard GeoJSON coordinates.

## Container Images

Image variants are available:

Image | Content | Zoom | Size
:-------------------------------: | :----------------------- | :-------: | :--------:
`ghcr.io/wooymasta/dzmap:latest`  | Only binaries and config | —         | `~15 MB`
`ghcr.io/wooymasta/dzmap:vanilla` | Official game + DLC maps | **Lvl 6** | `~200 MB`
`ghcr.io/wooymasta/dzmap:slim`    | Official + modded (40+)  | **Lvl 4** | `~170 MB`
`ghcr.io/wooymasta/dzmap:full`    | Official + modded (40+)  | **Lvl 6** | `~1.6 GB`

## Components

### Loader (`cmd/loader`)

The core utility for fetching and processing map data.

* Downloads tiles from remote sources or slices local single-file images
  (TIFF, BMP, PNG) into XYZ tiles.
* Normalizes all tiles to WebP format.
* Fetches location data ([xam.nu]/[iZurvive]) and converts it to standard
  GeoJSON (WGS84 Lat/Lon).
* Supports concurrent downloads and specific map filtering.

### Server (`cmd/server`)

A lightweight, high-performance HTTP server written in Go.

* Serves tiles and GeoJSON with ETag caching.
* Provides a simple Leaflet-based web viewer.
* Exposes a JSON API (`/api/maps`) listing available maps and their
  metadata.
* Handles missing tiles by serving a transparent 1x1 image.

### Config to GeoJSON (`cmd/cfg2json`)

A CLI utility to parse DayZ `cfgNames.hpp` files and convert class
definitions into GeoJSON. This allows mission developers to easily export
custom locations.

## Configuration

Configuration is handled via `config.yaml`. Example:

```yaml
zoom: 7
maps:
  - name: chernarusplus
    size: 15360
    # Template for remote tiles
    topographic: https://static.xam.nu/dayz/maps/chernarusplus/1.27/topographic/{z}/{x}/{y}.webp
    locations: https://static.xam.nu/dayz/json/chernarusplus/1.28-2.json

  - name: utes
    size: 5120
    # Single source file (will be sliced and converted)
    satellite: ./sources/utes_sat.tif
    tile_size: 256
```

## Usage

### Loader

Run the loader to populate the `maps/` directory.

```bash
# Process all maps defined in config
./loader -c config.yaml

# Process specific maps only
./loader -c config.yaml --limit chernarusplus --limit namalsk

# Force overwrite existing files
./loader -f
```

### Server

Serve the processed data.

```bash
./server -c config.yaml --addr 0.0.0.0 --port 8080
```

Access the map viewer at `http://localhost:8080`.

### Cfg2Json

Convert C++ header definitions to GeoJSON.

```bash
# Read from file, output to file
./cfg2json --size 15360 --in cfgNames.hpp --out locations.geojson

# Pipe input
cat cfgNames.hpp | ./cfg2json --size 12800 > locations.json
```

## API & Standards

* **Coordinates:** All location data is converted to WGS84
  (Latitude/Longitude).
* **Tile Layer:** Served at `/maps/{mapName}/{layer}/{z}/{x}/{y}.webp`.
* **GeoJSON:** Served at `/maps/{mapName}/locations.geojson`.
* **Map Config:** Available at `/api/maps`.

<!-- links -->
[MetricZ]: https://github.com/WoozyMasta/metricz
[#114371]: https://github.com/grafana/grafana/pull/114371
[xam.nu]: https://dayz.xam.nu
[iZurvive]: https://izurvive.com
