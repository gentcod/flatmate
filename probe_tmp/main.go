// Throwaway live probe: one real T1 ExtractAnswers call against the
// configured Meta Model API endpoint. Run once, verify, delete.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/gentcod/flatmate/internal/ai"
	"github.com/gentcod/flatmate/internal/util"
)

const sample = `I'm an early riser who keeps things spotless and can't stand a messy kitchen. ` +
	`Cigarette smoke is a total dealbreaker for me. I work from home three days a week ` +
	`and I like a quiet flat, though I'm happy to split bills evenly.`

func main() {
	cfg, err := util.LoadConfig(".env")
	if err != nil {
		fmt.Println("config:", err)
		os.Exit(1)
	}
	if !cfg.HasModel() {
		fmt.Println("model not configured")
		os.Exit(1)
	}
	llm, err := ai.NewHTTPModel(cfg.ModelEndpoint, cfg.ModelName, cfg.ModelToken)
	if err != nil {
		fmt.Println("model:", err)
		os.Exit(1)
	}
	drafts, err := ai.ExtractAnswers(context.Background(), llm, sample)
	if err != nil {
		fmt.Println("extract:", err)
		os.Exit(1)
	}
	fmt.Printf("got %d drafts (model %s):\n", len(drafts), cfg.ModelName)
	for _, d := range drafts {
		fmt.Printf("- %s = %q hard=%v weight=%d\n", d.Dimension, d.Value, d.Hard, d.Weight)
	}
}
