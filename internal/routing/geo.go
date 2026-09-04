package routing

import (
	"math"
	"motopath-generator/internal/domain"
)

const (
	EarthRadiusMeters = 6371000.0 // Earth's mean radius in meters
	DegToRad          = math.Pi / 180.0
	RadToDeg          = 180.0 / math.Pi
)

// HaversineDistance computes the great-circle distance between two GPS points in meters.
func HaversineDistance(p1, p2 domain.GeoPoint) float64 {
	dLat := (p2.Lat - p1.Lat) * DegToRad
	dLon := (p2.Lon - p1.Lon) * DegToRad

	lat1 := p1.Lat * DegToRad
	lat2 := p2.Lat * DegToRad

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLon/2)*math.Sin(dLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return EarthRadiusMeters * c
}

// DestinationPoint calculates a new GeoPoint given start point, bearing in degrees (0 = North, 90 = East), and distance in meters.
func DestinationPoint(start domain.GeoPoint, bearingDeg float64, distanceMeters float64) domain.GeoPoint {
	brng := bearingDeg * DegToRad
	dDivR := distanceMeters / EarthRadiusMeters

	lat1 := start.Lat * DegToRad
	lon1 := start.Lon * DegToRad

	lat2 := math.Asin(math.Sin(lat1)*math.Cos(dDivR) +
		math.Cos(lat1)*math.Sin(dDivR)*math.Cos(brng))

	lon2 := lon1 + math.Atan2(math.Sin(brng)*math.Sin(dDivR)*math.Cos(lat1),
		math.Cos(dDivR)-math.Sin(lat1)*math.Sin(lat2))

	return domain.GeoPoint{
		Lat: lat2 * RadToDeg,
		Lon: lon2 * RadToDeg,
	}
}

// Bearing calculates the initial bearing (forward azimuth) from p1 to p2 in degrees [0, 360).
func Bearing(p1, p2 domain.GeoPoint) float64 {
	lat1 := p1.Lat * DegToRad
	lat2 := p2.Lat * DegToRad
	dLon := (p2.Lon - p1.Lon) * DegToRad

	y := math.Sin(dLon) * math.Cos(lat2)
	x := math.Cos(lat1)*math.Sin(lat2) - math.Sin(lat1)*math.Cos(lat2)*math.Cos(dLon)
	brng := math.Atan2(y, x) * RadToDeg

	return math.Mod(brng+360.0, 360.0)
}

// BoundingBox computes [minLat, minLon, maxLat, maxLon] around a center point with a radius in meters.
func BoundingBox(center domain.GeoPoint, radiusMeters float64) [4]float64 {
	// Latitude offset: 1 deg lat ~= 111,139 meters
	latOffset := (radiusMeters / EarthRadiusMeters) * RadToDeg
	// Longitude offset adjusts for latitude
	lonOffset := (radiusMeters / (EarthRadiusMeters * math.Cos(center.Lat*DegToRad))) * RadToDeg

	return [4]float64{
		center.Lat - latOffset,
		center.Lon - lonOffset,
		center.Lat + latOffset,
		center.Lon + lonOffset,
	}
}
