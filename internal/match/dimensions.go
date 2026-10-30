// Package match is the pure-Go matching engine: no network, no LLM,
// no clock. This file defines the questionnaire dimensions.
package match

import "strconv"

// Dimension types.
const (
	Scale  = "scale"
	Choice = "choice"
)

// Groups for weighted scoring.
const (
	GroupLifestyle     = "lifestyle"
	GroupInterpersonal = "interpersonal"
	GroupModifier      = "modifier" // smoking/pets: lifestyle modifiers
)

// Dimension describes one questionnaire axis.
type Dimension struct {
	Name    string
	Type    string   // Scale or Choice
	Min     int      // scales only
	Max     int      // scales only
	Choices []string // choices only
	Group   string
}

// Dimensions lists all 12 questionnaire axes in display order.
var Dimensions = []Dimension{
	{Name: "cleanliness", Type: Scale, Min: 1, Max: 5, Group: GroupLifestyle},
	{Name: "noise", Type: Scale, Min: 1, Max: 5, Group: GroupLifestyle},
	{Name: "sleep", Type: Choice, Choices: []string{"early", "flex", "late"}, Group: GroupLifestyle},
	{Name: "guests", Type: Scale, Min: 1, Max: 5, Group: GroupLifestyle},
	{Name: "privacy", Type: Scale, Min: 1, Max: 5, Group: GroupLifestyle},
	{Name: "social", Type: Choice, Choices: []string{"quiet", "mixed", "social"}, Group: GroupLifestyle},
	{Name: "expenses", Type: Choice, Choices: []string{"equal", "itemized", "mixed"}, Group: GroupInterpersonal},
	{Name: "disagreements", Type: Choice, Choices: []string{"talk", "cooldown", "write"}, Group: GroupInterpersonal},
	{Name: "smoking", Type: Choice, Choices: []string{"no", "outside", "yes"}, Group: GroupModifier},
	{Name: "pets", Type: Choice, Choices: []string{"none", "ok", "have"}, Group: GroupModifier},
	{Name: "wfh", Type: Scale, Min: 0, Max: 5, Group: GroupLifestyle},
	{Name: "employment", Type: Choice, Choices: []string{"employed", "self-employed", "student", "unemployed"}, Group: GroupLifestyle},
}

// ByName returns the dimension definition, if known.
func ByName(name string) (Dimension, bool) {
	for _, d := range Dimensions {
		if d.Name == name {
			return d, true
		}
	}
	return Dimension{}, false
}

// ValidValue reports whether value is legal for the named dimension.
func ValidValue(name, value string) bool {
	d, ok := ByName(name)
	if !ok {
		return false
	}
	if d.Type == Choice {
		for _, c := range d.Choices {
			if value == c {
				return true
			}
		}
		return false
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return false
	}
	return n >= d.Min && n <= d.Max
}
