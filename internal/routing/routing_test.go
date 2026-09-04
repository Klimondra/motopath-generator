package routing

import (
	"testing"

	"motopath-generator/internal/domain"
)

func TestHaversineDistance(t *testing.T) {
	// Prague Old Town Square to Prague Castle (~1.5 km)
	p1 := domain.GeoPoint{Lat: 50.0875, Lon: 14.4214}
	p2 := domain.GeoPoint{Lat: 50.0906, Lon: 14.4005}

	dist := HaversineDistance(p1, p2)
	if dist < 1300 || dist > 1700 {
		t.Errorf("Expected distance ~1500m, got %.2fm", dist)
	}
}

func TestDestinationPoint(t *testing.T) {
	start := domain.GeoPoint{Lat: 50.0, Lon: 14.0}
	dest := DestinationPoint(start, 90.0, 1000.0) // 1km East

	dist := HaversineDistance(start, dest)
	if dist < 990 || dist > 1010 {
		t.Errorf("Expected ~1000m, got %.2fm", dist)
	}

	if dest.Lon <= start.Lon {
		t.Errorf("Expected dest.Lon > start.Lon, got %.5f vs %.5f", dest.Lon, start.Lon)
	}
}

func TestDijkstraWithPenalties(t *testing.T) {
	// Build a simple 4-node diamond graph:
	//     1 (Start)
	//    / \
	//   2   3
	//    \ /
	//     4 (Target)
	g := NewGraph()
	g.Nodes[1] = &Node{ID: 1, Point: domain.GeoPoint{Lat: 50.0, Lon: 14.0}}
	g.Nodes[2] = &Node{ID: 2, Point: domain.GeoPoint{Lat: 50.01, Lon: 13.99}}
	g.Nodes[3] = &Node{ID: 3, Point: domain.GeoPoint{Lat: 50.01, Lon: 14.01}}
	g.Nodes[4] = &Node{ID: 4, Point: domain.GeoPoint{Lat: 50.02, Lon: 14.0}}

	e12 := &Edge{ID: 1, From: 1, To: 2, DistanceM: 100, BaseCost: 100, Tags: map[string]string{"highway": "track"}}
	e24 := &Edge{ID: 2, From: 2, To: 4, DistanceM: 100, BaseCost: 100, Tags: map[string]string{"highway": "track"}}
	e13 := &Edge{ID: 3, From: 1, To: 3, DistanceM: 100, BaseCost: 105, Tags: map[string]string{"highway": "track"}}
	e34 := &Edge{ID: 4, From: 3, To: 4, DistanceM: 100, BaseCost: 105, Tags: map[string]string{"highway": "track"}}

	g.Adjacency[1] = []*Edge{e12, e13}
	g.Adjacency[2] = []*Edge{e24}
	g.Adjacency[3] = []*Edge{e34}

	// 1st run without penalties: path 1 -> 2 -> 4 should be preferred (cost 200 vs 210)
	res1, err := g.ShortestPath(1, 4, nil)
	if err != nil {
		t.Fatalf("Path 1 failed: %v", err)
	}
	if len(res1.Edges) != 2 || res1.Edges[0].ID != 1 || res1.Edges[1].ID != 2 {
		t.Errorf("Expected path 1-2-4, got edges: %+v", res1.Edges)
	}

	// 2nd run with heavy penalty on edge 1 and 2: path 1 -> 3 -> 4 should be taken
	penalties := map[int64]float64{1: 50.0, 2: 50.0}
	res2, err := g.ShortestPath(1, 4, penalties)
	if err != nil {
		t.Fatalf("Path 2 failed: %v", err)
	}
	if len(res2.Edges) != 2 || res2.Edges[0].ID != 3 || res2.Edges[1].ID != 4 {
		t.Errorf("Expected path 1-3-4 with penalties, got edges: %+v", res2.Edges)
	}
}
