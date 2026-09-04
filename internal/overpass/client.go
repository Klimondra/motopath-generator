package overpass

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"sync"
	"time"

	"motopath-generator/internal/domain"
)

var defaultEndpoints = []string{
	"https://overpass-api.de/api/interpreter",
	"https://lz4.overpass-api.de/api/interpreter",
	"https://overpass.kumi.systems/api/interpreter",
	"https://maps.mail.ru/osm/tools/overpass/api/interpreter",
}

type cacheEntry struct {
	data      *domain.OSMResponse
	timestamp time.Time
}

// Client handles Overpass API communication with caching and automatic failover.
type Client struct {
	httpClient *http.Client
	endpoints  []string
	cache      map[string]cacheEntry
	cacheMu    sync.RWMutex
}

// NewClient returns a new configured Overpass client.
func NewClient(timeout time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		endpoints: defaultEndpoints,
		cache:     make(map[string]cacheEntry),
	}
}

// makeCacheKey generates a localized spatial cache key.
func makeCacheKey(lat, lon, radiusM float64) string {
	// Round to ~100m grid for smart local caching
	rLat := math.Round(lat*500) / 500
	rLon := math.Round(lon*500) / 500
	rRad := math.Ceil(radiusM/500) * 500
	return fmt.Sprintf("%.4f:%.4f:%.0f", rLat, rLon, rRad)
}

// FetchOsmData queries Overpass API for road/track network within radius meters around center.
func (c *Client) FetchOsmData(ctx context.Context, center domain.GeoPoint, radiusM float64) (*domain.OSMResponse, error) {
	cacheKey := makeCacheKey(center.Lat, center.Lon, radiusM)

	c.cacheMu.RLock()
	if entry, found := c.cache[cacheKey]; found {
		if time.Since(entry.timestamp) < 30*time.Minute {
			c.cacheMu.RUnlock()
			log.Printf("[Overpass] Cache hit for key %s", cacheKey)
			return entry.data, nil
		}
	}
	c.cacheMu.RUnlock()

	// Build Overpass QL query:
	// Includes off-road tracks, paths, service roads, unclassified, residential, and connecting tertiary roads
	query := fmt.Sprintf(`[out:json][timeout:30];
(
  way["highway"~"^(track|unclassified|residential|tertiary|service|path|living_street)$"](around:%d,%.6f,%.6f);
);
out body;
>;
out skel qt;`, int(radiusM), center.Lat, center.Lon)

	var lastErr error
	for _, endpoint := range c.endpoints {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		log.Printf("[Overpass] Querying endpoint %s (radius: %.0fm)", endpoint, radiusM)
		respData, err := c.executeOverpassQuery(ctx, endpoint, query)
		if err != nil {
			log.Printf("[Overpass] Endpoint %s failed: %v", endpoint, err)
			lastErr = err
			continue
		}

		if len(respData.Elements) == 0 {
			lastErr = fmt.Errorf("empty response from %s", endpoint)
			continue
		}

		// Store in cache
		c.cacheMu.Lock()
		c.cache[cacheKey] = cacheEntry{
			data:      respData,
			timestamp: time.Now(),
		}
		c.cacheMu.Unlock()

		return respData, nil
	}

	return nil, fmt.Errorf("all Overpass endpoints failed: %w", lastErr)
}

func (c *Client) executeOverpassQuery(ctx context.Context, endpoint, query string) (*domain.OSMResponse, error) {
	reqBody := url.Values{}
	reqBody.Set("data", query)

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBufferString(reqBody.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "MotoPathGenerator/1.0 (DIY GoKart Route Planner)")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodySnippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(bodySnippet))
	}

	var osmResp domain.OSMResponse
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&osmResp); err != nil {
		return nil, fmt.Errorf("failed to decode Overpass JSON: %w", err)
	}

	return &osmResp, nil
}
