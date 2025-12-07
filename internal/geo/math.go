package geo

import "math"

// GameToMetricZ converts Game Coordinates (0..Size) to World WGS84 (Lon/Lat)
// using a Mercator projection adapted for the game map size.
//
// It maps the game world (0 to mapSize) to the longitude range [-180, 180]
// and applies an inverse Mercator projection for latitude.
func GameToMetricZ(x, z, mapSize float64) (lon, lat float64) {
	// x: [0..size] -> lon: [-180..180]
	longitudeScale := 360.0 / mapSize
	lon = x*longitudeScale - 180.0

	// z: [0..size] -> mercatorY: [-PI..PI]
	mercatorScale := (2.0 * math.Pi) / mapSize
	mercatorY := z*mercatorScale - math.Pi

	// Inverse Mercator projection
	latRad := (2.0 * math.Atan(math.Exp(mercatorY))) - (math.Pi * 0.5)

	const MaxLat = 85.05112878
	lat = latRad * (180.0 / math.Pi)

	if lat > MaxLat {
		lat = MaxLat
	} else if lat < -MaxLat {
		lat = -MaxLat
	}

	return lon, lat
}
