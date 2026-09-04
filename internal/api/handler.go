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

// Handler handles HTTP requests for route generation.
type Handler struct {
	overpassClient *overpass.Client
}

// NewHandler creates a new API handler.
func NewHandler(opClient *overpass.Client) *Handler {
	return &Handler{
		overpassClient: opClient,
	}
}

// RegisterRoutes registers all HTTP endpoints on the standard net/http mux (Go 1.22+).
func (h *Handler) RegisterRoutes(mux *http.ServeMux, staticDir string) {
	// API routes
	mux.HandleFunc("GET /api/health", h.handleHealth)
	mux.HandleFunc("POST /api/route", h.handleGenerateRoute)

	// Static files
	fs := http.FileServer(http.Dir(staticDir))
	mux.Handle("GET /", fs)
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "ok",
		"service": "motopath-generator",
		"time":    time.Now().Format(time.RFC3339),
	})
}

func (h *Handler) handleGenerateRoute(w http.ResponseWriter, r *http.Request) {
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

	// Compute search radius around user (at least 2.5km, up to ~18km depending on desired distance)
	searchRadiusM := (req.TargetDistKm * 1000.0) / (2.0 * math.Pi) * 1.7
	if searchRadiusM < 2500 {
		searchRadiusM = 2500
	}
	if searchRadiusM > 18000 {
		searchRadiusM = 18000
	}

	log.Printf("[API] Request loop: Start=(%.5f, %.5f), TargetDist=%.1f km, Radius=%.0f m, Strictness=%s",
		req.StartLat, req.StartLon, req.TargetDistKm, searchRadiusM, req.Strictness)

	// 1. Fetch OSM road/track data
	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()

	osmData, err := h.overpassClient.FetchOsmData(ctx, startPoint, searchRadiusM)
	if err != nil {
		log.Printf("[API] Overpass fetch error: %v", err)
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

	log.Printf("[API] Built graph: %d nodes, %d edges", len(graph.Nodes), len(graph.Edges))

	// 3. Generate off-road circuit
	generator := routing.NewCircuitGenerator(graph, costCfg)
	routeResp, err := generator.GenerateCircuit(startPoint, req.TargetDistKm, req.Seed)
	if err != nil {
		log.Printf("[API] Circuit generation error: %v", err)
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
