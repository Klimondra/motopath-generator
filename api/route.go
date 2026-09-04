package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"time"

	"motopath-generator/internal/domain"
	"motopath-generator/internal/overpass"
	"motopath-generator/internal/routing"
)

// Shared Overpass client (reused across warm serverless invocations for caching)
var opClient = overpass.NewClient(35 * time.Second)

// Handler is the Vercel Serverless Function entry point for /api/route.
func Handler(w http.ResponseWriter, r *http.Request) {
	// CORS Headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Health check / GET status
	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":  "ok",
			"service": "motopath-generator",
			"time":    time.Now().Format(time.RFC3339),
		})
		return
	}

	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method Not Allowed. Use POST to generate a route or GET for health check.")
		return
	}

	var req domain.RouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload: "+err.Error())
		return
	}

	// Validate inputs
	if req.StartLat < -90 || req.StartLat > 90 || req.StartLon < -180 || req.StartLon > 180 {
		writeError(w, http.StatusBadRequest, "Invalid GPS coordinates")
		return
	}
	if req.TargetDistKm < 1.0 {
		req.TargetDistKm = 5.0
	} else if req.TargetDistKm > 50.0 {
		req.TargetDistKm = 50.0
	}
	if req.Strictness == "" {
		req.Strictness = "balanced"
	}

	startPoint := domain.GeoPoint{
		Lat: req.StartLat,
		Lon: req.StartLon,
	}

	// Compute search radius around user
	searchRadiusM := (req.TargetDistKm * 1000.0) / (2.0 * math.Pi) * 1.7
	if searchRadiusM < 2500 {
		searchRadiusM = 2500
	}
	if searchRadiusM > 18000 {
		searchRadiusM = 18000
	}

	log.Printf("[Vercel API] Request loop: Start=(%.5f, %.5f), TargetDist=%.1f km, Radius=%.0f m, Strictness=%s",
		req.StartLat, req.StartLon, req.TargetDistKm, searchRadiusM, req.Strictness)

	// 1. Fetch OSM road/track data
	ctx, cancel := context.WithTimeout(r.Context(), 40*time.Second)
	defer cancel()

	osmData, err := opClient.FetchOsmData(ctx, startPoint, searchRadiusM)
	if err != nil {
		log.Printf("[Vercel API] Overpass fetch error: %v", err)
		writeError(w, http.StatusBadGateway, fmt.Sprintf("Failed to fetch map data from OpenStreetMap: %v", err))
		return
	}

	// 2. Build graph with cost weights
	costCfg := routing.GetCostConfig(req.Strictness)
	graph := routing.BuildGraphFromOSM(osmData, costCfg)

	if len(graph.Nodes) < 10 || len(graph.Edges) < 5 {
		writeError(w, http.StatusUnprocessableEntity, "Not enough trails or roads found in the selected area. Try a larger distance or different starting spot.")
		return
	}

	log.Printf("[Vercel API] Built graph: %d nodes, %d edges", len(graph.Nodes), len(graph.Edges))

	// 3. Generate off-road circuit
	generator := routing.NewCircuitGenerator(graph, costCfg)
	routeResp, err := generator.GenerateCircuit(startPoint, req.TargetDistKm, req.Seed)
	if err != nil {
		log.Printf("[Vercel API] Circuit generation error: %v", err)
		writeError(w, http.StatusUnprocessableEntity, fmt.Sprintf("Could not find a valid closed off-road loop: %v. Try adjusting distance or strictness.", err))
		return
	}

	writeJSON(w, http.StatusOK, routeResp)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"error":   message,
	})
}
