package processor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/woozymasta/dzmap/internal/geo"
)

// Internal structures for JSON parsing
type izurviveLocation struct {
	NameEN string  `json:"nameEN"`
	Type   string  `json:"type"`
	Lat    float64 `json:"lat"`
	Lng    float64 `json:"lng"`
}

// fetchIzurvive downloads and parses location data from iZurvive format.
func fetchIzurvive(client *http.Client, url string) (geo.GeoJSONFeatureCollection, error) {
	resp, err := client.Get(url)
	if err != nil {
		return geo.GeoJSONFeatureCollection{}, err
	}
	// Explicitly ignore close error as it's a read-only operation
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		return geo.GeoJSONFeatureCollection{}, fmt.Errorf("status %d", resp.StatusCode)
	}

	var locs []izurviveLocation
	if err := json.NewDecoder(resp.Body).Decode(&locs); err != nil {
		return geo.GeoJSONFeatureCollection{}, err
	}

	fc := geo.GeoJSONFeatureCollection{Type: "FeatureCollection", Features: []geo.GeoJSONFeature{}}
	for _, l := range locs {
		fc.Features = append(fc.Features, geo.GeoJSONFeature{
			Type: "Feature",
			Geometry: geo.GeoJSONGeometry{
				Type:        "Point",
				Coordinates: []float64{l.Lng, l.Lat},
			},
			Properties: map[string]interface{}{
				"name": l.NameEN,
				"type": strings.ToLower(l.Type),
			},
		})
	}

	return fc, nil
}
