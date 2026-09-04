package routing

import (
	"container/heap"
	"fmt"
	"math"
)

// priorityItem represents an entry in Dijkstra's priority queue.
type priorityItem struct {
	nodeID int64
	cost   float64
	index  int
}

type priorityQueue []*priorityItem

func (pq priorityQueue) Len() int           { return len(pq) }
func (pq priorityQueue) Less(i, j int) bool { return pq[i].cost < pq[j].cost }
func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}
func (pq *priorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*priorityItem)
	item.index = n
	*pq = append(*pq, item)
}
func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*pq = old[0 : n-1]
	return item
}

// PathResult contains the computed path nodes, edges, and distances.
type PathResult struct {
	Nodes        []*Node
	Edges        []*Edge
	TotalCost    float64
	TotalDistance float64
}

// ShortestPath finds the lowest-cost path from startNodeID to targetNodeID using Dijkstra's algorithm.
// edgePenalties maps Edge.ID -> penalty multiplier (e.g., 50.0 for already traversed edges).
func (g *Graph) ShortestPath(startNodeID, targetNodeID int64, edgePenalties map[int64]float64) (*PathResult, error) {
	if startNodeID == targetNodeID {
		startNode, ok := g.Nodes[startNodeID]
		if !ok {
			return nil, fmt.Errorf("start node %d not found", startNodeID)
		}
		return &PathResult{
			Nodes:        []*Node{startNode},
			Edges:        []*Edge{},
			TotalCost:    0,
			TotalDistance: 0,
		}, nil
	}

	dist := make(map[int64]float64)
	prevNode := make(map[int64]int64)
	prevEdge := make(map[int64]*Edge)

	for id := range g.Nodes {
		dist[id] = math.MaxFloat64
	}
	dist[startNodeID] = 0

	pq := make(priorityQueue, 0)
	heap.Init(&pq)
	heap.Push(&pq, &priorityItem{nodeID: startNodeID, cost: 0})

	visited := make(map[int64]bool)

	for pq.Len() > 0 {
		current := heap.Pop(&pq).(*priorityItem)
		u := current.nodeID

		if visited[u] {
			continue
		}
		visited[u] = true

		if u == targetNodeID {
			break
		}

		for _, edge := range g.Adjacency[u] {
			v := edge.To
			if visited[v] {
				continue
			}

			// Apply dynamic edge penalty
			penalty := 1.0
			if edgePenalties != nil {
				if p, ok := edgePenalties[edge.ID]; ok {
					penalty = p
				}
			}

			effectiveCost := edge.BaseCost * penalty
			newCost := dist[u] + effectiveCost

			if newCost < dist[v] {
				dist[v] = newCost
				prevNode[v] = u
				prevEdge[v] = edge
				heap.Push(&pq, &priorityItem{nodeID: v, cost: newCost})
			}
		}
	}

	if dist[targetNodeID] == math.MaxFloat64 {
		return nil, fmt.Errorf("no path found between node %d and %d", startNodeID, targetNodeID)
	}

	// Reconstruct path
	var pathNodes []*Node
	var pathEdges []*Edge
	curr := targetNodeID
	totalRealDist := 0.0

	for curr != startNodeID {
		pathNodes = append([]*Node{g.Nodes[curr]}, pathNodes...)
		e := prevEdge[curr]
		if e != nil {
			pathEdges = append([]*Edge{e}, pathEdges...)
			totalRealDist += e.DistanceM
		}
		curr = prevNode[curr]
	}
	pathNodes = append([]*Node{g.Nodes[startNodeID]}, pathNodes...)

	return &PathResult{
		Nodes:        pathNodes,
		Edges:        pathEdges,
		TotalCost:    dist[targetNodeID],
		TotalDistance: totalRealDist,
	}, nil
}
