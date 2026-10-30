package match

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
)

// Weights tunes group influence. Loaded from weights.yaml, never hard-coded.
type Weights struct {
	Practical     float64
	Lifestyle     float64
	Interpersonal float64
	DiscussCap    int
}

// LoadWeights parses a minimal `key: value` YAML subset.
// Unknown keys and sums other than 1.0 are errors.
func LoadWeights(path string) (Weights, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Weights{}, err
	}
	w := Weights{DiscussCap: 60}
	seen := map[string]bool{}
	for i, line := range strings.Split(string(raw), "\n") {
		if cut := strings.Index(line, "#"); cut >= 0 {
			line = line[:cut]
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		kv := strings.SplitN(line, ":", 2)
		if len(kv) != 2 {
			return Weights{}, fmt.Errorf("weights line %d: want key: value", i+1)
		}
		key := strings.TrimSpace(kv[0])
		val := strings.TrimSpace(kv[1])
		seen[key] = true
		switch key {
		case "practical", "lifestyle", "interpersonal":
			f, err := strconv.ParseFloat(val, 64)
			if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
				return Weights{}, fmt.Errorf("weights line %d: bad number", i+1)
			}
			if f < 0 || f > 1 {
				return Weights{}, fmt.Errorf("weights line %d: out of range 0–1", i+1)
			}
			switch key {
			case "practical":
				w.Practical = f
			case "lifestyle":
				w.Lifestyle = f
			case "interpersonal":
				w.Interpersonal = f
			}
		case "discuss_cap":
			n, err := strconv.Atoi(val)
			if err != nil || n < 0 || n > 100 {
				return Weights{}, fmt.Errorf("weights line %d: bad discuss_cap", i+1)
			}
			w.DiscussCap = n
		default:
			return Weights{}, fmt.Errorf("weights line %d: unknown key %q", i+1, key)
		}
	}
	for _, k := range []string{"practical", "lifestyle", "interpersonal"} {
		if !seen[k] {
			return Weights{}, fmt.Errorf("weights: missing %q", k)
		}
	}
	if math.Abs(w.Practical+w.Lifestyle+w.Interpersonal-1) > 0.001 {
		return Weights{}, fmt.Errorf("weights: must sum to 1.0")
	}
	return w, nil
}
