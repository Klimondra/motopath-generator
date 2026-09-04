package routing

import (
	"strings"
)

// CostConfig holds highway and surface weight multipliers.
type CostConfig struct {
	TrackMultiplier        float64
	PathMultiplier         float64
	ServiceMultiplier      float64
	UnclassifiedMultiplier float64
	ResidentialMultiplier  float64
	TertiaryMultiplier     float64
	DefaultMultiplier      float64
	UnpavedBonus           float64
	PavedPenalty           float64
}

// GetCostConfig returns tuning parameters based on chosen strictness profile.
func GetCostConfig(strictness string) CostConfig {
	switch strings.ToLower(strictness) {
	case "hardcore":
		return CostConfig{
			TrackMultiplier:        1.0,
			PathMultiplier:         1.2,
			ServiceMultiplier:      8.0,
			UnclassifiedMultiplier: 35.0,
			ResidentialMultiplier:  50.0,
			TertiaryMultiplier:     120.0,
			DefaultMultiplier:      100.0,
			UnpavedBonus:           0.8,
			PavedPenalty:           2.0,
		}
	case "relaxed":
		return CostConfig{
			TrackMultiplier:        1.0,
			PathMultiplier:         1.2,
			ServiceMultiplier:      3.0,
			UnclassifiedMultiplier: 6.0,
			ResidentialMultiplier:  10.0,
			TertiaryMultiplier:     20.0,
			DefaultMultiplier:      30.0,
			UnpavedBonus:           0.95,
			PavedPenalty:           1.2,
		}
	case "balanced":
		fallthrough
	default:
		return CostConfig{
			TrackMultiplier:        1.0,
			PathMultiplier:         1.5,
			ServiceMultiplier:      6.0,
			UnclassifiedMultiplier: 18.0,
			ResidentialMultiplier:  25.0,
			TertiaryMultiplier:     50.0,
			DefaultMultiplier:      60.0,
			UnpavedBonus:           0.85,
			PavedPenalty:           1.5,
		}
	}
}

// CalculateEdgeCost computes the effective weighted cost of traversing an edge.
func CalculateEdgeCost(distanceM float64, tags map[string]string, cfg CostConfig) float64 {
	highway := strings.ToLower(tags["highway"])
	surface := strings.ToLower(tags["surface"])
	tracktype := strings.ToLower(tags["tracktype"])

	multiplier := cfg.DefaultMultiplier

	switch highway {
	case "track":
		multiplier = cfg.TrackMultiplier
		// Higher grade tracks (grade3, grade4, grade5) are rougher/dirtier off-road tracks
		switch tracktype {
		case "grade2", "grade3", "grade4":
			multiplier *= 0.9
		case "grade1": // paved or compacted track
			multiplier *= 1.1
		}
	case "path":
		multiplier = cfg.PathMultiplier
	case "service":
		multiplier = cfg.ServiceMultiplier
	case "unclassified":
		multiplier = cfg.UnclassifiedMultiplier
	case "residential", "living_street":
		multiplier = cfg.ResidentialMultiplier
	case "tertiary", "tertiary_link":
		multiplier = cfg.TertiaryMultiplier
	}

	// Surface adjustments
	switch surface {
	case "dirt", "earth", "ground", "grass", "gravel", "fine_gravel", "sand", "unpaved", "mud", "compacted":
		multiplier *= cfg.UnpavedBonus
	case "asphalt", "paved", "concrete", "paving_stones", "sett":
		multiplier *= cfg.PavedPenalty
	}

	// Ensure multiplier is at least 0.1
	if multiplier < 0.1 {
		multiplier = 0.1
	}

	return distanceM * multiplier
}

// IsOffroad determines if the highway/surface tags represent an off-road track.
func IsOffroad(tags map[string]string) bool {
	highway := strings.ToLower(tags["highway"])
	surface := strings.ToLower(tags["surface"])

	if highway == "track" || highway == "path" {
		return surface != "asphalt" && surface != "paved" && surface != "concrete"
	}

	switch surface {
	case "dirt", "earth", "ground", "grass", "gravel", "fine_gravel", "sand", "unpaved", "mud":
		return true
	}

	return false
}

// IsPaved determines if the road is paved.
func IsPaved(tags map[string]string) bool {
	surface := strings.ToLower(tags["surface"])
	switch surface {
	case "asphalt", "paved", "concrete", "paving_stones", "sett":
		return true
	}

	highway := strings.ToLower(tags["highway"])
	if highway == "residential" || highway == "tertiary" || highway == "secondary" || highway == "primary" || highway == "living_street" {
		return surface == "" || surface == "asphalt" || surface == "paved"
	}

	return false
}
