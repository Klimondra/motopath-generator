package routing

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"motopath-generator/internal/domain"
)

// CircuitGenerator orchestrates the closed loop track generation.
type CircuitGenerator struct {
	graph *Graph
	cfg   CostConfig
}

// NewCircuitGenerator creates a new generator instance.
func NewCircuitGenerator(g *Graph, cfg CostConfig) *CircuitGenerator {
	return &CircuitGenerator{
		graph: g,
		cfg:   cfg,
	}
}

type candidateCircuit struct {
	segments   []domain.RouteSegment
	points     []domain.GeoPoint
	stats      domain.RouteStats
	score      float64
	totalDistM float64
}

// GenerateCircuit creates an off-road loop around startPoint of target length targetDistKm.
func (cg *CircuitGenerator) GenerateCircuit(startPoint domain.GeoPoint, targetDistKm float64, seed int64) (*domain.RouteResponse, error) {
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	r := rand.New(rand.NewSource(seed))

	// Find starting node in graph
	startNode, distToStart, err := cg.graph.FindNearestNode(startPoint, 5000, true, nil)
	if err != nil {
		return nil, fmt.Errorf("could not find accessible track near start position: %w", err)
	}

	// Compute reachable nodes from startNode
	reachableSet := cg.graph.GetReachableNodes(startNode.ID)
	if len(reachableSet) < 15 {
		// Start node might be on a dead stub; try finding a node belonging to a larger connected component
		allNodes := make([]int64, 0, len(cg.graph.Nodes))
		for id := range cg.graph.Nodes {
			if len(cg.graph.Adjacency[id]) > 0 {
				allNodes = append(allNodes, id)
			}
		}

		var bestStart *Node
		maxCompSize := 0
		for _, nid := range allNodes {
			comp := cg.graph.GetReachableNodes(nid)
			if len(comp) > maxCompSize {
				maxCompSize = len(comp)
				bestStart = cg.graph.Nodes[nid]
				reachableSet = comp
			}
		}
		if bestStart != nil && maxCompSize >= 15 {
			startNode = bestStart
			distToStart = HaversineDistance(startPoint, startNode.Point)
		}
	}

	targetDistMeters := targetDistKm * 1000.0
	// Target radius calculation
	baseRadius := targetDistMeters / (2.4 * math.Pi)
	if baseRadius < 400 {
		baseRadius = 400
	}

	var bestCircuit *candidateCircuit
	attempts := 24
	waypointCounts := []int{3, 4, 3, 2, 4, 3}

	for i := 0; i < attempts; i++ {
		numWaypoints := waypointCounts[i%len(waypointCounts)]
		baseAngle := r.Float64() * 360.0
		clockwise := r.Intn(2) == 0
		radiusScale := 0.75 + r.Float64()*0.5 // 0.75x to 1.25x

		circuit, err := cg.tryBuildCircuit(startNode, baseRadius*radiusScale, numWaypoints, baseAngle, clockwise, r, targetDistMeters, reachableSet)
		if err != nil {
			continue
		}

		distRatio := circuit.totalDistM / targetDistMeters
		if distRatio > 2.5 || distRatio < 0.25 {
			continue // Too distorted
		}

		distAccuracy := 1.0 - math.Abs(1.0-distRatio)
		// Combined score: offroad ratio (60%) + distance accuracy (40%)
		score := (circuit.stats.OffroadPercent * 0.6) + (distAccuracy * 40.0)

		if bestCircuit == nil || score > bestCircuit.score {
			circuit.score = score
			bestCircuit = circuit
		}
	}

	if bestCircuit == nil {
		return nil, fmt.Errorf("failed to generate closed loop circuit. Please try a different distance or start point.")
	}

	// Calculate bounding box
	minLat, minLon := 90.0, 180.0
	maxLat, maxLon := -90.0, -180.0
	for _, pt := range bestCircuit.points {
		if pt.Lat < minLat {
			minLat = pt.Lat
		}
		if pt.Lat > maxLat {
			maxLat = pt.Lat
		}
		if pt.Lon < minLon {
			minLon = pt.Lon
		}
		if pt.Lon > maxLon {
			maxLon = pt.Lon
		}
	}

	return &domain.RouteResponse{
		Success:      true,
		Message:      fmt.Sprintf("Closed loop generated successfully (nearest trail: %.0fm from start)", distToStart),
		Points:       bestCircuit.points,
		Segments:     bestCircuit.segments,
		Stats:        bestCircuit.stats,
		TargetRadius: baseRadius,
		BoundingBox:  [4]float64{minLat, minLon, maxLat, maxLon},
	}, nil
}

func (cg *CircuitGenerator) tryBuildCircuit(startNode *Node, radius float64, numWaypoints int, baseAngle float64, clockwise bool, r *rand.Rand, targetDistMeters float64, reachableSet map[int64]bool) (*candidateCircuit, error) {
	// 1. Select waypoints on a radial circle around start node
	waypoints := make([]*Node, 0, numWaypoints)
	step := 360.0 / float64(numWaypoints+1)
	if !clockwise {
		step = -step
	}

	for i := 1; i <= numWaypoints; i++ {
		angle := baseAngle + float64(i)*step + (r.Float64()*25.0 - 12.5)
		jitteredRadius := radius * (0.8 + r.Float64()*0.4)

		targetPt := DestinationPoint(startNode.Point, angle, jitteredRadius)
		wpNode, _, err := cg.graph.FindNearestNode(targetPt, radius*1.5, true, reachableSet)
		if err != nil {
			continue
		}
		if wpNode.ID != startNode.ID {
			waypoints = append(waypoints, wpNode)
		}
	}

	if len(waypoints) < 1 {
		return nil, fmt.Errorf("not enough reachable waypoints found")
	}

	// 2. Sequential routing with dynamic visited-edge penalties
	visitedEdges := make(map[int64]float64)
	visitedNodes := make(map[int64]int)
	var allEdges []*Edge
	var allPoints []domain.GeoPoint
	var segments []domain.RouteSegment

	legs := append([]*Node{startNode}, waypoints...)
	legs = append(legs, startNode) // Close loop back to start

	allPoints = append(allPoints, startNode.Point)
	visitedNodes[startNode.ID]++

	for i := 0; i < len(legs)-1; i++ {
		fromNode := legs[i]
		toNode := legs[i+1]

		pathRes, err := cg.graph.ShortestPath(fromNode.ID, toNode.ID, visitedEdges)
		if err != nil {
			return nil, fmt.Errorf("leg %d failed: %w", i, err)
		}

		for _, edge := range pathRes.Edges {
			allEdges = append(allEdges, edge)
			// Heavy penalty (80x) on edge and its reverse to prevent retracing
			visitedEdges[edge.ID] = 80.0
			for _, revEdge := range cg.graph.Adjacency[edge.To] {
				if revEdge.To == edge.From {
					visitedEdges[revEdge.ID] = 80.0
				}
			}

			toPt := cg.graph.Nodes[edge.To].Point
			allPoints = append(allPoints, toPt)
			visitedNodes[edge.To]++

			fromPt := cg.graph.Nodes[edge.From].Point
			segments = append(segments, domain.RouteSegment{
				FromNodeID: edge.From,
				ToNodeID:   edge.To,
				DistanceM:  edge.DistanceM,
				Highway:    edge.Highway,
				Surface:    edge.Surface,
				TrackType:  edge.TrackType,
				Name:       edge.Name,
				Points:     []domain.GeoPoint{fromPt, toPt},
			})
		}
	}

	if len(allEdges) == 0 {
		return nil, fmt.Errorf("no edges in generated circuit")
	}

	// Check backtracking ratio
	revisitedCount := 0
	for _, count := range visitedNodes {
		if count > 2 {
			revisitedCount++
		}
	}
	revisitRatio := float64(revisitedCount) / float64(len(visitedNodes))
	if revisitRatio > 0.50 {
		return nil, fmt.Errorf("circuit contains excessive backtracking (ratio: %.2f)", revisitRatio)
	}

	// 3. Compute detailed statistics
	stats := calculateStats(allEdges, len(waypoints))

	return &candidateCircuit{
		segments:   segments,
		points:     allPoints,
		stats:      stats,
		totalDistM: stats.TotalDistanceM,
	}, nil
}

func calculateStats(edges []*Edge, waypointCount int) domain.RouteStats {
	var totalDist float64
	var offroadDist float64
	var pavedDist float64
	hwBreakdown := make(map[string]float64)
	surfBreakdown := make(map[string]float64)

	for _, e := range edges {
		totalDist += e.DistanceM

		hw := e.Highway
		if hw == "" {
			hw = "unknown"
		}
		hwBreakdown[hw] += e.DistanceM

		surf := e.Surface
		if surf == "" {
			if hw == "track" || hw == "path" {
				surf = "unpaved (inferred)"
			} else {
				surf = "asphalt (inferred)"
			}
		}
		surfBreakdown[surf] += e.DistanceM

		if IsOffroad(e.Tags) {
			offroadDist += e.DistanceM
		}
		if IsPaved(e.Tags) {
			pavedDist += e.DistanceM
		}
	}

	offroadPercent := 0.0
	pavedPercent := 0.0
	if totalDist > 0 {
		offroadPercent = (offroadDist / totalDist) * 100.0
		pavedPercent = (pavedDist / totalDist) * 100.0
	}

	estimatedTimeMin := int(math.Round((totalDist / 1000.0) / 22.0 * 60.0))
	if estimatedTimeMin < 1 {
		estimatedTimeMin = 1
	}

	return domain.RouteStats{
		TotalDistanceM:   math.Round(totalDist),
		TotalDistanceKm:  math.Round((totalDist/1000.0)*100) / 100,
		OffroadDistanceM: math.Round(offroadDist),
		OffroadPercent:   math.Round(offroadPercent*10) / 10,
		PavedDistanceM:   math.Round(pavedDist),
		PavedPercent:     math.Round(pavedPercent*10) / 10,
		HighwayBreakdown: hwBreakdown,
		SurfaceBreakdown: surfBreakdown,
		EstimatedTimeMin: estimatedTimeMin,
		WaypointCount:    waypointCount,
	}
}
