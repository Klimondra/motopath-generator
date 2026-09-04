package domain

// GeoPoint represents a geographic coordinate with latitude and longitude.
type GeoPoint struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// RouteRequest defines the input payload for generating a route.
type RouteRequest struct {
	StartLat     float64 `json:"start_lat"`
	StartLon     float64 `json:"start_lon"`
	TargetDistKm float64 `json:"target_dist_km"` // Desired total loop distance in kilometers (e.g., 5.0, 10.0)
	Strictness   string  `json:"strictness"`     // "hardcore", "balanced", "relaxed"
	Seed         int64   `json:"seed,omitempty"` // Optional random seed for variety
}

// RouteSegment represents a part of the computed route with surface metadata.
type RouteSegment struct {
	FromNodeID int64    `json:"from_node_id"`
	ToNodeID   int64    `json:"to_node_id"`
	DistanceM  float64  `json:"distance_m"`
	Highway    string   `json:"highway"`
	Surface    string   `json:"surface"`
	TrackType  string   `json:"track_type"`
	Name       string   `json:"name,omitempty"`
	Points     []GeoPoint `json:"points"`
}

// RouteStats contains summary analytics for the generated route.
type RouteStats struct {
	TotalDistanceM   float64            `json:"total_distance_m"`
	TotalDistanceKm  float64            `json:"total_distance_km"`
	OffroadDistanceM float64            `json:"offroad_distance_m"`
	OffroadPercent   float64            `json:"offroad_percent"`
	PavedDistanceM   float64            `json:"paved_distance_m"`
	PavedPercent     float64            `json:"paved_percent"`
	HighwayBreakdown map[string]float64 `json:"highway_breakdown_m"` // Highway type -> length in meters
	SurfaceBreakdown map[string]float64 `json:"surface_breakdown_m"` // Surface type -> length in meters
	EstimatedTimeMin int                `json:"estimated_time_min"`  // Assuming ~25 km/h off-road average
	WaypointCount    int                `json:"waypoint_count"`
}

// RouteResponse is the complete response sent to the frontend.
type RouteResponse struct {
	Success      bool           `json:"success"`
	Message      string         `json:"message,omitempty"`
	Points       []GeoPoint     `json:"points"`
	Segments     []RouteSegment `json:"segments"`
	Stats        RouteStats     `json:"stats"`
	TargetRadius float64        `json:"target_radius_m"`
	BoundingBox  [4]float64     `json:"bounding_box"` // [minLat, minLon, maxLat, maxLon]
}

// OSMNode represents a single point parsed from Overpass API.
type OSMNode struct {
	ID   int64   `json:"id"`
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
	Tags map[string]string `json:"tags,omitempty"`
}

// OSMWay represents a road/track consisting of multiple nodes.
type OSMWay struct {
	ID    int64             `json:"id"`
	Nodes []int64           `json:"nodes"`
	Tags  map[string]string `json:"tags"`
}

// OSMResponse is the raw JSON structure returned by Overpass API.
type OSMResponse struct {
	Elements []struct {
		Type  string            `json:"type"`
		ID    int64             `json:"id"`
		Lat   float64           `json:"lat"`
		Lon   float64           `json:"lon"`
		Nodes []int64           `json:"nodes"`
		Tags  map[string]string `json:"tags"`
	} `json:"elements"`
}
