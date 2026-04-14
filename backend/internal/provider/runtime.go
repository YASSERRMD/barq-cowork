package provider

import (
	"regexp"
	"strconv"
	"strings"
)

var modelSizePattern = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)b`)

// SuggestedMaxConcurrentRequests returns a conservative per-provider request
// concurrency cap so the runtime does not oversubscribe hosted APIs or small
// local models.
func SuggestedMaxConcurrentRequests(cfg ProviderConfig) int {
	switch strings.ToLower(strings.TrimSpace(cfg.ProviderName)) {
	case "zai":
		return 3
	case "ollama":
		if IsWeakLocalModel(cfg.Model) {
			return 1
		}
		return 2
	default:
		return 6
	}
}

// IsWeakLocalModel flags small local models that tend to struggle with long
// agent prompts, multi-turn tool loops, and large JSON payloads.
func IsWeakLocalModel(model string) bool {
	size := modelParameterBillions(model)
	if size > 0 && size <= 8.5 {
		return true
	}

	name := strings.ToLower(strings.TrimSpace(model))
	for _, marker := range []string{
		"0.5b", "1.5b", "3b", "7b", "8b",
		"mini", "tiny", "small",
	} {
		if strings.Contains(name, marker) {
			return true
		}
	}
	return false
}

func modelParameterBillions(model string) float64 {
	match := modelSizePattern.FindStringSubmatch(strings.ToLower(model))
	if len(match) < 2 {
		return 0
	}
	size, err := strconv.ParseFloat(match[1], 64)
	if err != nil {
		return 0
	}
	return size
}
