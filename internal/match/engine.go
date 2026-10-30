// Eligibility, scoring, and explanations. Pure: no I/O, no clock reads.
package match

import (
	"math"
	"strconv"
	"strings"
	"time"
)

// Response is one user's answer to a dimension.
type Response struct {
	Value  string
	Hard   bool
	Weight int // 1–3
}

// Candidate is the engine's view of one user.
type Candidate struct {
	UserID       int64
	AreaText     string
	HasCoords    bool
	Lat, Lng     float64
	Relocating   bool
	BudgetMin    int64
	BudgetMax    int64
	MoveEarliest string // YYYY-MM-DD
	MoveLatest   string
	Gender       string // woman|man|nonbinary ("" = unset)
	GenderPref   string // anyone|women|men|nonbinary ("" = anyone)
	Answers      map[string]Response
}

// DimensionResult scores one axis, per CONTEXT.md §10.
type DimensionResult struct {
	Name    string
	Score   int
	Weight  float64
	Aligned bool
	Discuss bool
	Note    string
}

// Breakdown is the engine→UI→Muse contract.
type Breakdown struct {
	Total         int
	Practical     int
	Lifestyle     int
	Interpersonal int
	Dimensions    []DimensionResult
	Alignments    []string
	Differences   []string
	Discuss       []string
}

// Eligible reports whether a and b may match at all.
// Modes, budgets, dates, and areas never gate; only identity,
// blocks, explicit hard requirements, and gender preferences do.
func Eligible(a, b Candidate, blockedAB, blockedBA bool) bool {
	if a.UserID == b.UserID {
		return false
	}
	if blockedAB || blockedBA {
		return false
	}
	return !violatesHard(a, b) && !violatesHard(b, a) &&
		!genderExcluded(a, b) && !genderExcluded(b, a)
}

// violatesHard reports whether b breaks any hard requirement of a.
func violatesHard(a, b Candidate) bool {
	for _, d := range Dimensions {
		ra, ok := a.Answers[d.Name]
		if !ok || !ra.Hard {
			continue
		}
		rb, ok := b.Answers[d.Name]
		if !ok {
			continue // unanswered: scores later, never gates
		}
		if !hardCompatible(d, ra.Value, rb.Value) {
			return true
		}
	}
	return false
}

// genderExcluded reports whether other's gender falls outside wanter's
// stated preference. Anyone (or unset) accepts all; unset gender satisfies
// only anyone, since a specific preference can't be verified.
func genderExcluded(wanter, other Candidate) bool {
	switch wanter.GenderPref {
	case "", "anyone":
		return false
	}
	switch other.Gender {
	case "woman":
		return wanter.GenderPref != "women"
	case "man":
		return wanter.GenderPref != "men"
	case "nonbinary":
		return wanter.GenderPref != "nonbinary"
	}
	return true
}

// bridgeValue is the tolerant option that adapts to either side.
func bridgeValue(dim string) string {
	switch dim {
	case "sleep":
		return "flex"
	case "social":
		return "mixed"
	case "expenses":
		return "mixed"
	case "smoking":
		return "outside"
	case "pets":
		return "ok"
	}
	return ""
}

// hardCompatible reports whether otherVal satisfies a hard requirement.
// Choices need equality or a bridging side; scales allow a gap of 1.
func hardCompatible(d Dimension, hardVal, otherVal string) bool {
	if hardVal == otherVal {
		return true
	}
	if b := bridgeValue(d.Name); b != "" && (hardVal == b || otherVal == b) {
		return true
	}
	if d.Type == Scale {
		av, aerr := strconv.Atoi(hardVal)
		bv, berr := strconv.Atoi(otherVal)
		if aerr != nil || berr != nil {
			return false
		}
		return abs(av-bv) <= 1
	}
	return false
}

// Score rates the pair 0–100 and explains why.
func Score(a, b Candidate, w Weights) Breakdown {
	var bd Breakdown
	var lSum, lW, iSum, iW float64
	for _, d := range Dimensions {
		dr := scoreDimension(d, a.Answers[d.Name], b.Answers[d.Name], w.DiscussCap)
		bd.Dimensions = append(bd.Dimensions, dr)
		switch d.Group {
		case GroupLifestyle:
			lSum += float64(dr.Score) * dr.Weight
			lW += dr.Weight
		case GroupInterpersonal:
			iSum += float64(dr.Score) * dr.Weight
			iW += dr.Weight
		case GroupModifier: // half-weighted into lifestyle
			lSum += float64(dr.Score) * dr.Weight * 0.5
			lW += dr.Weight * 0.5
		}
	}
	budget := scoreBudget(a, b)
	area := scoreArea(a, b)
	dates := scoreDates(a, b)
	bd.Dimensions = append(bd.Dimensions,
		practicalResult("budget", budget, "Strong alignment on budget", "budgets differ"),
		practicalResult("area", area, "Strong alignment on area", "areas differ"),
		practicalResult("dates", dates, "Move-in windows overlap well", "move-in windows barely overlap"),
	)
	bd.Practical = int(math.Round(float64(budget+area+dates) / 3))
	if lW > 0 {
		bd.Lifestyle = int(math.Round(lSum / lW))
	}
	if iW > 0 {
		bd.Interpersonal = int(math.Round(iSum / iW))
	}
	bd.Total = int(math.Round(w.Practical*float64(bd.Practical) +
		w.Lifestyle*float64(bd.Lifestyle) + w.Interpersonal*float64(bd.Interpersonal)))
	for _, dr := range bd.Dimensions {
		switch {
		case dr.Discuss:
			bd.Discuss = append(bd.Discuss, "Talk about "+dr.Note)
		case dr.Aligned:
			bd.Alignments = append(bd.Alignments, dr.Note)
		default:
			bd.Differences = append(bd.Differences, dr.Note)
		}
	}
	return bd
}

func scoreDimension(d Dimension, ra, rb Response, cap int) DimensionResult {
	dr := DimensionResult{Name: d.Name, Weight: float64(ra.Weight+rb.Weight) / 2}
	if dr.Weight < 1 {
		dr.Weight = 1
	}
	if d.Type == Scale {
		av, aerr := strconv.Atoi(ra.Value)
		bv, berr := strconv.Atoi(rb.Value)
		if aerr != nil || berr != nil {
			dr.Score = 0
			dr.Note = d.Name + ": missing answer"
			return dr
		}
		gap := abs(av - bv)
		span := d.Max - d.Min
		if span <= 0 {
			span = 1
		}
		dr.Score = 100 - gap*100/span
		if float64(gap)/float64(span) >= 0.5 {
			dr.Discuss = true
		}
	} else {
		switch {
		case ra.Value == rb.Value && ra.Value != "":
			dr.Score = 100
		case ra.Value == "" || rb.Value == "":
			dr.Score = 0
		case bridgePair(d.Name, ra.Value, rb.Value):
			dr.Score = 70
		default:
			dr.Score = 30
			dr.Discuss = true
		}
	}
	if dr.Discuss && dr.Score > cap {
		dr.Score = cap
	}
	dr.Aligned = dr.Score >= 75 && !dr.Discuss
	if dr.Aligned {
		dr.Note = "Strong match on " + d.Name
	} else {
		dr.Note = d.Name + ": " + displayVal(ra.Value) + " vs " + displayVal(rb.Value)
	}
	return dr
}

func bridgePair(dim, x, y string) bool {
	b := bridgeValue(dim)
	return b != "" && (x == b || y == b)
}

func displayVal(v string) string {
	if v == "" {
		return "—"
	}
	return v
}

func practicalResult(name string, score int, alignedNote, diffNote string) DimensionResult {
	dr := DimensionResult{Name: name, Score: score, Weight: 1, Aligned: score >= 75}
	if dr.Aligned {
		dr.Note = alignedNote
	} else {
		dr.Note = diffNote
	}
	return dr
}

// scoreBudget rates range overlap 0–100. Never gates.
func scoreBudget(a, b Candidate) int {
	lo := max(a.BudgetMin, b.BudgetMin)
	hi := min(a.BudgetMax, b.BudgetMax)
	spanLo := min(a.BudgetMin, b.BudgetMin)
	spanHi := max(a.BudgetMax, b.BudgetMax)
	if spanHi <= spanLo {
		return 100
	}
	overlap := max(hi-lo, 0)
	return int(math.Round(100 * float64(overlap) / float64(spanHi-spanLo)))
}

// scoreArea rates location fit. Same area text wins; relocating users
// match on area text only; otherwise shared coords give a distance bonus.
func scoreArea(a, b Candidate) int {
	ta := normalizeArea(a.AreaText)
	tb := normalizeArea(b.AreaText)
	if ta != "" && ta == tb {
		return 100
	}
	if a.Relocating || b.Relocating || !a.HasCoords || !b.HasCoords {
		return 40
	}
	s := 100 - haversine(a.Lat, a.Lng, b.Lat, b.Lng)*2
	return int(math.Round(max(s, 0)))
}

func normalizeArea(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// scoreDates rates move-in overlap against the longer window. Never gates.
func scoreDates(a, b Candidate) int {
	as, err1 := time.Parse("2006-01-02", a.MoveEarliest)
	ae, err2 := time.Parse("2006-01-02", a.MoveLatest)
	bs, err3 := time.Parse("2006-01-02", b.MoveEarliest)
	be, err4 := time.Parse("2006-01-02", b.MoveLatest)
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
		return 0
	}
	overlap := daysBetween(maxTime(as, bs), minTime(ae, be))
	span := max(daysBetween(as, ae), daysBetween(bs, be))
	if span <= 0 {
		return 100
	}
	return int(math.Round(100 * max(overlap, 0) / span))
}

func daysBetween(from, to time.Time) float64 {
	return to.Sub(from).Hours()/24 + 1
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func haversine(lat1, lng1, lat2, lng2 float64) float64 {
	const r = 6371 // km
	dLat := (lat2 - lat1) * math.Pi / 180
	dLng := (lng2 - lng1) * math.Pi / 180
	h := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * r * math.Asin(math.Sqrt(h))
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
