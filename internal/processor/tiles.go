package processor

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/woozymasta/dzmap/internal/config"

	"github.com/chai2010/webp"
	"github.com/rs/zerolog/log"
	_ "golang.org/x/image/bmp"
	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

// TileCoordinate represents a specific tile.
type TileCoordinate struct {
	Z, X, Y int
}

type job struct {
	URLTemplate string
	BaseDir     string
	Coord       TileCoordinate
}

type result struct {
	Coord TileCoordinate
	Valid bool
}

// ProcessTiles handles the downloading or slicing of map tiles.
// It supports both downloading from a URL template and slicing from a single large image.
func ProcessTiles(client *http.Client, m config.Map, concurrency, defaultZoom int, force, fastCheck bool) {
	types := map[string]string{
		"topographic": m.Topographic,
		"satellite":   m.Satellite,
	}

	zoomLimit := m.ZoomLimit
	if zoomLimit <= 0 {
		zoomLimit = defaultZoom
	}

	for typeName, source := range types {
		if source == "" {
			continue
		}

		baseDir := filepath.Join("maps", m.Name, typeName)

		// Fast Check
		if fastCheck {
			if _, err := os.Stat(baseDir); err == nil {
				log.Info().
					Str("map", m.Name).
					Str("layer", typeName).
					Msg("Layer directory exists, skipping (fast-check)")

				continue
			}
		}

		// Detect if source is a template or a single file
		if strings.Contains(source, "{z}") || strings.Contains(source, "{x}") {
			// --- Standard Download Mode ---
			processDownloadMode(client, source, m.Name, typeName, zoomLimit, concurrency, force)
		} else {
			// --- Single Image Slicing Mode ---
			log.Info().
				Str("map", m.Name).
				Str("layer", typeName).
				Str("source", source).
				Msg("Starting single image processing (download & slice)")

			tileSize := m.TileSize
			if tileSize <= 0 {
				tileSize = 256
			}

			if err := processSingleImage(client, source, baseDir, zoomLimit, tileSize, force); err != nil {
				log.Error().Err(err).Str("map", m.Name).Msg("Failed to process single image")
			}
		}
	}
}

// processDownloadMode handles the standard downloading of pre-tiled maps.
func processDownloadMode(client *http.Client, urlTemplate, mapName, typeName string, zoomLimit, concurrency int, force bool) {
	log.Info().
		Str("map", mapName).
		Str("layer", typeName).
		Msg("Starting tile download")

	currentLevelTiles := []TileCoordinate{{0, 0, 0}}

	for z := 0; z <= zoomLimit; z++ {
		if len(currentLevelTiles) == 0 {
			break
		}
		if z > 0 && !probeLevel(client, currentLevelTiles, urlTemplate) {
			log.Info().Int("zoom", z).Msg("No data found at zoom level, stopping")
			break
		}

		log.Debug().Int("zoom", z).Int("count", len(currentLevelTiles)).Msg("Processing zoom level")

		validTiles := processBatch(client, concurrency, currentLevelTiles, urlTemplate, mapName, typeName, force)

		nextLevelTiles := make([]TileCoordinate, 0, len(validTiles)*4)
		for _, t := range validTiles {
			nx, ny := t.X*2, t.Y*2
			nextLevelTiles = append(nextLevelTiles,
				TileCoordinate{Z: z + 1, X: nx, Y: ny},
				TileCoordinate{Z: z + 1, X: nx + 1, Y: ny},
				TileCoordinate{Z: z + 1, X: nx, Y: ny + 1},
				TileCoordinate{Z: z + 1, X: nx + 1, Y: ny + 1},
			)
		}
		currentLevelTiles = nextLevelTiles
	}
}

// processSingleImage downloads/opens a large image and slices it into tiles.
func processSingleImage(client *http.Client, sourceURL, baseDir string, zoomLimit, tileSize int, force bool) error {
	// Load the source image (Download or Local File)
	srcImg, err := loadSourceImage(client, sourceURL)
	if err != nil {
		return err
	}

	bounds := srcImg.Bounds()
	log.Info().
		Int("width", bounds.Dx()).
		Int("height", bounds.Dy()).
		Msg("Source image loaded, starting tiling")

	// Iterate through Zoom Levels
	for z := 0; z <= zoomLimit; z++ {
		// Calculate dimensions for this zoom level
		// Grid size: 2^z
		gridSize := 1 << z
		totalPixels := gridSize * tileSize

		log.Debug().
			Int("zoom", z).
			Int("grid", gridSize).
			Int("px", totalPixels).
			Msg("Processing zoom level")

		// Resize Source Image to fit the grid
		// We use BiLinear as a good balance between speed and quality for maps.
		// For huge downscaling (e.g. 20k -> 256px), CatmullRom is better but slower.
		// Since we resize from Original every time, quality is preserved.
		dstImg := image.NewRGBA(image.Rect(0, 0, totalPixels, totalPixels))
		xdraw.CatmullRom.Scale(dstImg, dstImg.Bounds(), srcImg, srcImg.Bounds(), draw.Over, nil)

		// Slice and Save
		var wg sync.WaitGroup
		// Simple semaphore to limit file I/O concurrency
		sem := make(chan struct{}, 20)

		for x := 0; x < gridSize; x++ {
			for y := 0; y < gridSize; y++ {
				wg.Add(1)
				sem <- struct{}{}

				go func(zx, zy int) {
					defer wg.Done()
					defer func() { <-sem }()

					// Crop
					rect := image.Rect(zx*tileSize, zy*tileSize, (zx+1)*tileSize, (zy+1)*tileSize)
					subImg := dstImg.SubImage(rect)

					// Save
					outPath := filepath.Join(
						baseDir,
						fmt.Sprintf("%d", z),
						fmt.Sprintf("%d", zx),
						fmt.Sprintf("%d", zy)+".webp",
					)

					if !force {
						if info, err := os.Stat(outPath); err == nil && info.Size() > 0 {
							return
						}
					}

					if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
						log.Error().Err(err).Msg("Failed to create dir")
						return
					}

					f, err := os.Create(outPath)
					if err != nil {
						log.Error().Err(err).Msg("Failed to create file")
						return
					}
					defer func() { _ = f.Close() }()

					if err := webp.Encode(f, subImg, &webp.Options{Lossless: false, Quality: 85}); err != nil {
						log.Error().Err(err).Msg("Failed to encode webp")
					}
				}(x, y)
			}
		}
		wg.Wait()
	}

	return nil
}

func loadSourceImage(client *http.Client, source string) (image.Image, error) {
	var reader io.Reader

	if strings.HasPrefix(source, "http") {
		// Remote URL
		log.Info().Str("url", source).Msg("Downloading source image...")
		resp, err := client.Get(source)
		if err != nil {
			return nil, err
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("download failed: %d", resp.StatusCode)
		}

		// Need to buffer for decoding if stream doesn't support seek (some decoders need it)
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(bodyBytes)
	} else {
		f, err := os.Open(source)
		if err != nil {
			return nil, err
		}
		defer func() { _ = f.Close() }()

		reader = f
	}

	img, format, err := image.Decode(reader)
	if err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}

	log.Info().Str("format", format).Msg("Image decoded successfully")
	return img, nil
}

func processBatch(
	client *http.Client,
	concurrency int,
	tiles []TileCoordinate,
	urlTpl, mapName, mapType string,
	force bool,
) []TileCoordinate {

	jobs := make(chan job, len(tiles))
	results := make(chan result, len(tiles))
	baseDir := filepath.Join("maps", mapName, mapType)

	go func() {
		for _, t := range tiles {
			jobs <- job{Coord: t, URLTemplate: urlTpl, BaseDir: baseDir}
		}
		close(jobs)
	}()

	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				isValid, err := downloadAndConvert(client, j, force)
				// Retry logic could be added here if needed, keeping it simple for now
				if err != nil {
					log.Trace().
						Err(err).
						Str("url", buildURL(j.URLTemplate, j.Coord)).
						Msg("Failed to download tile")

				}
				results <- result{Coord: j.Coord, Valid: isValid}
			}
		}()
	}
	wg.Wait()
	close(results)

	var valid []TileCoordinate
	for res := range results {
		if res.Valid {
			valid = append(valid, res.Coord)
		}
	}

	return valid
}

func downloadAndConvert(client *http.Client, j job, force bool) (bool, error) {
	outPath := filepath.Join(
		j.BaseDir,
		fmt.Sprintf("%d", j.Coord.Z),
		fmt.Sprintf("%d", j.Coord.X),
		fmt.Sprintf("%d", j.Coord.Y)+".webp")

	// Check existence if not forcing overwrite
	if !force {
		if info, err := os.Stat(outPath); err == nil && info.Size() > 0 {
			return true, nil
		}
	}

	url := buildURL(j.URLTemplate, j.Coord)
	resp, err := client.Get(url)
	if err != nil {
		return false, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == 404 {
		log.Trace().Str("url", url).Msg("Tile not found (404)")
		return false, nil
	}
	if resp.StatusCode != 200 {
		return false, fmt.Errorf("status code %d", resp.StatusCode)
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	img, _, err := image.Decode(bytes.NewReader(bodyBytes))
	if err != nil {
		log.Trace().Err(err).Str("url", url).Msg("Failed to decode image")
		return false, nil // Not an image or corrupted
	}

	// Filter out empty/1px tiles often returned by map servers for OOB areas
	if img.Bounds().Dx() <= 1 {
		log.Trace().Err(err).Str("url", url).Msg("Filtered empty tile")
		return false, nil
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return false, err
	}

	outFile, err := os.Create(outPath)
	if err != nil {
		return false, err
	}
	defer func() { _ = outFile.Close() }()

	if err := webp.Encode(outFile, img, &webp.Options{Lossless: false, Quality: 80}); err != nil {
		return false, err
	}

	return true, nil
}

func buildURL(tpl string, c TileCoordinate) string {
	s := strings.ReplaceAll(tpl, "{z}", fmt.Sprintf("%d", c.Z))
	s = strings.ReplaceAll(s, "{x}", fmt.Sprintf("%d", c.X))
	s = strings.ReplaceAll(s, "{y}", fmt.Sprintf("%d", c.Y))

	if strings.Contains(s, "{tms_y}") {
		maxCoord := (1 << c.Z) - 1
		tmsY := maxCoord - c.Y
		s = strings.ReplaceAll(s, "{tms_y}", fmt.Sprintf("%d", tmsY))
	}

	return s
}

func probeLevel(client *http.Client, tiles []TileCoordinate, urlTpl string) bool {
	// Check a few points (start, middle, end) to see if the zoom level has data
	probes := []TileCoordinate{}
	if len(tiles) > 0 {
		probes = append(probes, tiles[0])
	}
	if len(tiles) > 10 {
		probes = append(probes, tiles[len(tiles)/2])
	}
	if len(tiles) > 1 {
		probes = append(probes, tiles[len(tiles)-1])
	}

	for _, p := range probes {
		if checkTileExists(client, buildURL(urlTpl, p)) {
			return true
		}
	}

	return false
}

func checkTileExists(client *http.Client, url string) bool {
	resp, err := client.Get(url)
	if err != nil || resp.StatusCode != 200 {
		return false
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	img, _, err := image.Decode(bytes.NewReader(body))
	if err != nil {
		return false
	}

	return img.Bounds().Dx() > 1
}
