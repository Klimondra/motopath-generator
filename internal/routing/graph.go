package routing

import (
	"fmt"
	"math"
	"strings"

	"motopath-generator/internal/domain"
)

// Edge represents a directed or undirected connection between two nodes in the routing graph.
type Edge struct {
	ID        int64
	From      int64
	To        int64
	DistanceM float64
	BaseCost  float64
	Highway   string
	Surface   string
	TrackType string
	Name      string
	WayID     int64
	Tags      map[string]string
}

// Node represents a geographic intersection or curve point in the graph.
type Node struct {
	ID    int64
	Point domain.GeoPoint
}

// Graph is the in-memory road/track network representation.
type Graph struct {
	Nodes     map[int64]*Node
	Edges     []*Edge
	Adjacency map[int64][]*Edge
}

// NewGraph instantiates an empty Graph.
func NewGraph() *Graph {
	return &Graph{
		Nodes:     make(map[int64]*Node),
		Edges:     make([]*Edge, 0),
		Adjacency: make(map[int64][]*Edge),
	}
}

// BuildGraphFromOSM constructs the graph from Overpass OSM elements.
func BuildGraphFromOSM(osm *domain.OSMResponse, cfg CostConfig) *Graph {
	g := NewGraph()

	// 1. Collect all node coordinates
	for _, el := range osm.Elements {
		if el.Type == "node" {
			g.Nodes[el.ID] = &Node{
				ID: el.ID,
				Point: domain.GeoPoint{
					Lat: el.Lat,
					Lon: el.Lon,
				},
			}
		}
	}

	edgeIDCounter := int64(1)

	// 2. Create edges from ways
	for _, el := range osm.Elements {
		if el.Type != "way" || len(el.Nodes) < 2 {
			continue
		}

		highway := el.Tags["highway"]
		surface := el.Tags["surface"]
		tracktype := el.Tags["tracktype"]
		name := el.Tags["name"]
		oneway := strings.ToLower(el.Tags["oneway"]) == "yes" || el.Tags["oneway"] == "1"

		// Connect consecutive nodes
		for i := 0; i < len(el.Nodes)-1; i++ {
			uID := el.Nodes[i]
			vID := el.Nodes[i+1]

			uNode, okU := g.Nodes[uID]
			vNode, okV := g.Nodes[vID]

			if !okU || !okV {
				continue
			}

			distM := HaversineDistance(uNode.Point, vNode.Point)
			if distM < 0.01 {
				continue // Skip zero-length duplicate edges
			}

			cost := CalculateEdgeCost(distM, el.Tags, cfg)

			edgeForward := &Edge{
				ID:        edgeIDCounter,
				From:      uID,
				To:        vID,
				DistanceM: distM,
				BaseCost:  cost,
				Highway:   highway,
				Surface:   surface,
				TrackType: tracktype,
				Name:      name,
				WayID:     el.ID,
				Tags:      el.Tags,
			}
			edgeIDCounter++
			g.Edges = append(g.Edges, edgeForward)
			g.Adjacency[uID] = append(g.Adjacency[uID], edgeForward)

			// If not strictly oneway, add reverse edge
			if !oneway {
				edgeBackward := &Edge{
					ID:        edgeIDCounter,
					From:      vID,
					To:        uID,
					DistanceM: distM,
					BaseCost:  cost,
					Highway:   highway,
					Surface:   surface,
					TrackType: tracktype,
					Name:      name,
					WayID:     el.ID,
					Tags:      el.Tags,
				}
				edgeIDCounter++
				g.Edges = append(g.Edges, edgeBackward)
				g.Adjacency[vID] = append(g.Adjacency[vID], edgeBackward)
			}
		}
	}

	return g
}

// GetReachableNodes returns a set of all node IDs reachable from startNodeID via BFS.
func (g *Graph) GetReachableNodes(startNodeID int64) map[int64]bool {
	reachable := make(map[int64]bool)
	queue := []int64{startNodeID}
	reachable[startNodeID] = true

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		for _, edge := range g.Adjacency[curr] {
			if !reachable[edge.To] {
				reachable[edge.To] = true
				queue = append(queue, edge.To)
			}
		}
	}

	return reachable
}

// FindNearestNode finds the closest graph node to a target coordinate within maxRadiusMeters.
// If reachableSet is non-nil, only nodes in the set are considered.
func (g *Graph) FindNearestNode(target domain.GeoPoint, maxRadiusMeters float64, preferOffroad bool, reachableSet map[int64]bool) (*Node, float64, error) {
	if len(g.Nodes) == 0 {
		return nil, 0, fmt.Errorf("graph contains no nodes")
	}

	var bestNode *Node
	minScoreDist := math.MaxFloat64
	bestActualDist := math.MaxFloat64

	for id, node := range g.Nodes {
		if reachableSet != nil && !reachableSet[id] {
			continue
		}

		edges := g.Adjacency[id]
		if len(edges) == 0 {
			continue
		}

		actualDist := HaversineDistance(target, node.Point)
		if maxRadiusMeters > 0 && actualDist > maxRadiusMeters {
			continue
		}

		scoreDist := actualDist
		if preferOffroad {
			hasOffroadEdge := false
			for _, e := range edges {
				if IsOffroad(e.Tags) {
					hasOffroadEdge = true
					break
				}
			}
			if hasOffroadEdge {
				scoreDist *= 0.7 // 30% preference bonus for offroad nodes
			}
		}

		if scoreDist < minScoreDist {
			minScoreDist = scoreDist
			bestActualDist = actualDist
			bestNode = node
		}
	}

	// If nothing found with maxRadiusMeters constraint, try without distance limit among valid nodes
	if bestNode == nil && maxRadiusMeters > 0 {
		return g.FindNearestNode(target, 0, preferOffroad, reachableSet)
	}

	if bestNode == nil {
		return nil, 0, fmt.Errorf("no connected nodes found matching criteria")
	}

	return bestNode, bestActualDist, nil
}
