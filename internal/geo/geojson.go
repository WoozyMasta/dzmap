// Package geo handles geographic data structures and coordinate conversions.
package geo

// GeoJSONFeatureCollection represents a collection of geographic features.
// It follows the standard GeoJSON structure.
type GeoJSONFeatureCollection struct {
	Type     string           `json:"type" yaml:"type"`
	Features []GeoJSONFeature `json:"features" yaml:"features"`
}

// GeoJSONFeature represents a single geographic feature with geometry and properties.
type GeoJSONFeature struct {
	Properties map[string]interface{} `json:"properties" yaml:"properties"`
	Type       string                 `json:"type" yaml:"type"`
	Geometry   GeoJSONGeometry        `json:"geometry" yaml:"geometry"`
}

// GeoJSONGeometry represents the geometry of a feature (Point, Polygon, etc.).
type GeoJSONGeometry struct {
	Type        string    `json:"type" yaml:"type"`
	Coordinates []float64 `json:"coordinates" yaml:"coordinates"` // [Lon, Lat]
}
