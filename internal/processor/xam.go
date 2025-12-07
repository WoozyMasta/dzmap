package processor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/woozymasta/dzmap/internal/geo"
)

// Internal structures for JSON parsing
type xamRoot struct {
	Markers struct {
		Locations []struct {
			Type  string    `json:"w"`
			Pos   []float64 `json:"p"`
			Names []string  `json:"s"`
		} `json:"locations"`
	} `json:"markers"`
}

// fetchXam downloads and parses location data from Xam format.
func fetchXam(client *http.Client, url string, mapSize int) (geo.GeoJSONFeatureCollection, error) {
	resp, err := client.Get(url)
	if err != nil {
		return geo.GeoJSONFeatureCollection{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		return geo.GeoJSONFeatureCollection{}, fmt.Errorf("status %d", resp.StatusCode)
	}

	var root xamRoot
	if err := json.NewDecoder(resp.Body).Decode(&root); err != nil {
		return geo.GeoJSONFeatureCollection{}, err
	}

	fc := geo.GeoJSONFeatureCollection{Type: "FeatureCollection", Features: []geo.GeoJSONFeature{}}

	for _, loc := range root.Markers.Locations {
		if len(loc.Pos) < 2 {
			continue
		}
		name := "Unknown"
		if len(loc.Names) > 0 {
			name = loc.Names[0]
		}

		// Xam to Game conversion
		xamY, xamX := loc.Pos[0], loc.Pos[1]
		gameX := (xamX * float64(mapSize)) / 256.0
		gameZ := ((256.0 + xamY) * float64(mapSize)) / 256.0

		wLon, wLat := geo.GameToMetricZ(gameX, gameZ, float64(mapSize))

		fc.Features = append(fc.Features, geo.GeoJSONFeature{
			Type: "Feature",
			Geometry: geo.GeoJSONGeometry{
				Type:        "Point",
				Coordinates: []float64{wLon, wLat},
			},
			Properties: map[string]interface{}{
				"name": name,
				"type": strings.ToLower(loc.Type),
			},
		})
	}

	return fc, nil
}
