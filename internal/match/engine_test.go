package match

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testCandidate(id int64) Candidate {
	return Candidate{
		UserID:       id,
		AreaText:     "Yaba, Lagos",
		BudgetMin:    400000,
		BudgetMax:    600000,
		MoveEarliest: "2026-12-01",
		MoveLatest:   "2027-02-01",
		Answers:      defaultAnswers(),
	}
}

func defaultAnswers() map[string]Response {
	m := map[string]Response{}
	for _, d := range Dimensions {
		r := Response{Weight: 1}
		switch d.Name {
		case "sleep":
			r.Value = "flex"
		case "social":
			r.Value = "mixed"
		case "expenses":
			r.Value = "mixed"
		case "disagreements":
			r.Value = "talk"
		case "smoking":
			r.Value = "no"
		case "pets":
			r.Value = "none"
		case "wfh":
			r.Value = "2"
		case "employment":
			r.Value = "employed"
		default:
			r.Value = "3"
		}
		m[d.Name] = r
	}
	return m
}

func withAnswer(c Candidate, dim, value string, hard bool, weight int) Candidate {
	cp := map[string]Response{}
	for k, v := range c.Answers {
		cp[k] = v
	}
	cp[dim] = Response{Value: value, Hard: hard, Weight: weight}
	c.Answers = cp
	return c
}

func testWeights() Weights {
	return Weights{Practical: 0.35, Lifestyle: 0.45, Interpersonal: 0.20, DiscussCap: 60}
}

func dimByName(bd Breakdown, name string) DimensionResult {
	for _, dr := range bd.Dimensions {
		if dr.Name == name {
			return dr
		}
	}
	return DimensionResult{}
}

func TestEligible(t *testing.T) {
	base := testCandidate(1)
	other := testCandidate(2)
	cases := []struct {
		name    string
		a, b    Candidate
		blockAB bool
		blockBA bool
		want    bool
	}{
		{"self", base, base, false, false, false},
		{"strangers", base, other, false, false, true},
		{"blockedAB", base, other, true, false, false},
		{"blockedBA", base, other, false, true, false},
		{"hard satisfied",
			withAnswer(base, "smoking", "no", true, 3),
			withAnswer(other, "smoking", "no", false, 1),
			false, false, true},
		{"hard violated",
			withAnswer(base, "smoking", "no", true, 3),
			withAnswer(other, "smoking", "yes", false, 1),
			false, false, false},
		{"hard violated reverse",
			withAnswer(base, "smoking", "yes", false, 1),
			withAnswer(other, "smoking", "no", true, 3),
			false, false, false},
		{"hard bridge ok",
			withAnswer(base, "sleep", "early", true, 3),
			withAnswer(other, "sleep", "flex", false, 1),
			false, false, true},
		{"hard clash",
			withAnswer(base, "sleep", "early", true, 3),
			withAnswer(other, "sleep", "late", false, 1),
			false, false, false},
		{"hard scale near ok",
			withAnswer(base, "cleanliness", "5", true, 3),
			withAnswer(other, "cleanliness", "4", false, 1),
			false, false, true},
		{"hard scale far",
			withAnswer(base, "cleanliness", "5", true, 3),
			withAnswer(other, "cleanliness", "3", false, 1),
			false, false, false},
		{"hard pets none vs have",
			withAnswer(base, "pets", "none", true, 3),
			withAnswer(other, "pets", "have", false, 1),
			false, false, false},
		{"hard pets none vs ok",
			withAnswer(base, "pets", "none", true, 3),
			withAnswer(other, "pets", "ok", false, 1),
			false, false, true},
		{"modes budgets never gate",
			Candidate{UserID: 1, AreaText: "Yaba", BudgetMin: 0, BudgetMax: 1,
				MoveEarliest: "2026-12-01", MoveLatest: "2026-12-02", Answers: defaultAnswers()},
			Candidate{UserID: 2, AreaText: "Lekki", BudgetMin: 999999, BudgetMax: 9999999,
				MoveEarliest: "2028-01-01", MoveLatest: "2028-06-01", Answers: defaultAnswers()},
			false, false, true},
	}
	for _, tc := range cases {
		if got := Eligible(tc.a, tc.b, tc.blockAB, tc.blockBA); got != tc.want {
			t.Errorf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}

func withGender(c Candidate, gender, pref string) Candidate {
	c.Gender, c.GenderPref = gender, pref
	return c
}

func TestEligibleGender(t *testing.T) {
	womanAny := withGender(testCandidate(1), "woman", "anyone")
	manAny := withGender(testCandidate(2), "man", "anyone")
	cases := []struct {
		name string
		a, b Candidate
		want bool
	}{
		{"anyone pair", womanAny, manAny, true},
		{"unset is anyone", testCandidate(1), testCandidate(2), true},
		{"mutual specific",
			withGender(testCandidate(1), "woman", "women"),
			withGender(testCandidate(2), "woman", "women"),
			true},
		{"one-sided specific ok",
			withGender(testCandidate(1), "woman", "women"),
			withGender(testCandidate(2), "woman", "anyone"),
			true},
		{"excluded by a",
			withGender(testCandidate(1), "woman", "women"),
			manAny,
			false},
		{"excluded by b",
			womanAny,
			withGender(testCandidate(2), "man", "men"),
			false},
		{"nonbinary pref",
			withGender(testCandidate(1), "nonbinary", "anyone"),
			withGender(testCandidate(2), "woman", "nonbinary"),
			true},
		{"unset gender vs specific pref",
			withGender(testCandidate(1), "", "anyone"),
			withGender(testCandidate(2), "woman", "women"),
			false},
	}
	for _, tc := range cases {
		if got := Eligible(tc.a, tc.b, false, false); got != tc.want {
			t.Errorf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}

func TestScoreIdentical(t *testing.T) {
	bd := Score(testCandidate(1), testCandidate(2), testWeights())
	if bd.Total != 100 || bd.Practical != 100 || bd.Lifestyle != 100 || bd.Interpersonal != 100 {
		t.Fatalf("identical pair should score 100s, got %+v", bd)
	}
	if len(bd.Dimensions) != len(Dimensions)+3 {
		t.Fatalf("want %d dimension results, got %d", len(Dimensions)+3, len(bd.Dimensions))
	}
	if len(bd.Discuss) != 0 || len(bd.Differences) != 0 {
		t.Fatalf("identical pair should have no discuss/differences: %+v", bd)
	}
	if len(bd.Alignments) != len(bd.Dimensions) {
		t.Fatalf("identical pair should align everywhere: %+v", bd)
	}
}

func TestScoreClash(t *testing.T) {
	a := testCandidate(1)
	b := testCandidate(2)
	b = withAnswer(b, "social", "social", false, 1)
	b = withAnswer(b, "sleep", "late", false, 1)
	b = withAnswer(b, "guests", "5", false, 1)
	b = withAnswer(b, "expenses", "itemized", false, 1)
	a = withAnswer(a, "social", "quiet", false, 1)
	a = withAnswer(a, "sleep", "early", false, 1)
	a = withAnswer(a, "guests", "1", false, 1)
	a = withAnswer(a, "expenses", "equal", false, 1)
	bd := Score(a, b, testWeights())

	if dr := dimByName(bd, "social"); dr.Score != 30 || !dr.Discuss {
		t.Errorf("social clash: got %+v", dr)
	}
	if dr := dimByName(bd, "guests"); dr.Score != 0 || !dr.Discuss {
		t.Errorf("guests gap: got %+v", dr)
	}
	if dr := dimByName(bd, "expenses"); dr.Score != 30 || !dr.Discuss {
		t.Errorf("expenses clash: got %+v", dr)
	}
	if len(bd.Discuss) < 4 {
		t.Errorf("want >=4 discuss prompts, got %v", bd.Discuss)
	}
	if bd.Total >= 90 {
		t.Errorf("clashing pair should score below 90, got %d", bd.Total)
	}
}

func TestScoreBridge(t *testing.T) {
	a := withAnswer(testCandidate(1), "sleep", "early", false, 1)
	b := withAnswer(testCandidate(2), "sleep", "flex", false, 1)
	bd := Score(a, b, testWeights())
	if dr := dimByName(bd, "sleep"); dr.Score != 70 || dr.Discuss {
		t.Errorf("bridge pair: got %+v", dr)
	}
}

func TestEmploymentScoring(t *testing.T) {
	a := withAnswer(testCandidate(1), "employment", "employed", false, 1)
	b := withAnswer(testCandidate(2), "employment", "employed", false, 1)
	if dr := dimByName(Score(a, b, testWeights()), "employment"); dr.Score != 100 || dr.Discuss || !dr.Aligned {
		t.Errorf("exact: got %+v", dr)
	}
	bd := Score(a, withAnswer(testCandidate(2), "employment", "student", false, 1), testWeights())
	if dr := dimByName(bd, "employment"); dr.Score != 30 || !dr.Discuss {
		t.Errorf("clash: got %+v", dr)
	}
	found := false
	for _, d := range bd.Discuss {
		if strings.Contains(d, "employment") {
			found = true
		}
	}
	if !found {
		t.Errorf("clash should prompt discussion: %v", bd.Discuss)
	}
	if !ValidValue("employment", "self-employed") || ValidValue("employment", "retired") {
		t.Error("employment values: want employed/self-employed/student/unemployed only")
	}
}

func TestScoreWeights(t *testing.T) {
	a, b := testCandidate(1), withAnswer(testCandidate(2), "social", "social", false, 1)
	a = withAnswer(a, "social", "quiet", false, 1)
	w := Weights{Practical: 1, DiscussCap: 60}
	bd := Score(a, b, w)
	if bd.Total != bd.Practical {
		t.Errorf("all-practical weights: total %d != practical %d", bd.Total, bd.Practical)
	}
}

func TestBudgetNeverGates(t *testing.T) {
	a := testCandidate(1)
	b := testCandidate(2)
	b.BudgetMin, b.BudgetMax = 5000000, 6000000
	if !Eligible(a, b, false, false) {
		t.Fatal("disjoint budgets must stay eligible")
	}
	bd := Score(a, b, testWeights())
	if dr := dimByName(bd, "budget"); dr.Score != 0 {
		t.Errorf("disjoint budgets should score 0, got %+v", dr)
	}
}

func TestArea(t *testing.T) {
	a := testCandidate(1)
	b := testCandidate(2)
	b.AreaText = "  yaba, LAGOS "
	if got := scoreArea(a, b); got != 100 {
		t.Errorf("same area text: got %d", got)
	}
	b.AreaText = "Surulere, Lagos"
	a.HasCoords, b.HasCoords = true, true
	a.Lat, a.Lng = 6.51, 3.37
	b.Lat, b.Lng = 6.52, 3.38 // ~1.5km away
	if got := scoreArea(a, b); got < 90 {
		t.Errorf("nearby coords: got %d", got)
	}
	b.Lat, b.Lng = 9.06, 7.49 // Abuja
	if got := scoreArea(a, b); got != 0 {
		t.Errorf("far coords: got %d", got)
	}
	a.Relocating = true
	if got := scoreArea(a, b); got != 40 {
		t.Errorf("relocating ignores coords: got %d", got)
	}
}

func TestDates(t *testing.T) {
	a := testCandidate(1)
	if got := scoreDates(a, testCandidate(2)); got != 100 {
		t.Errorf("same window: got %d", got)
	}
	b := testCandidate(2)
	b.MoveEarliest, b.MoveLatest = "2028-01-01", "2028-06-01"
	if got := scoreDates(a, b); got != 0 {
		t.Errorf("disjoint windows: got %d", got)
	}
	b.MoveEarliest, b.MoveLatest = "2027-01-01", "2027-03-01"
	if got := scoreDates(a, b); got != 51 {
		t.Errorf("partial overlap: got %d want 51", got)
	}
}

func TestLoadWeights(t *testing.T) {
	write := func(t *testing.T, body string) string {
		t.Helper()
		p := filepath.Join(t.TempDir(), "weights.yaml")
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	w, err := LoadWeights(write(t, "# comment\n\npractical: 0.35\nlifestyle: 0.45\ninterpersonal: 0.20\ndiscuss_cap: 60\n"))
	if err != nil {
		t.Fatal(err)
	}
	if w.Practical != 0.35 || w.Lifestyle != 0.45 || w.Interpersonal != 0.20 || w.DiscussCap != 60 {
		t.Errorf("bad parse: %+v", w)
	}
	w, err = LoadWeights(write(t, "practical: 0.5\nlifestyle: 0.5\ninterpersonal: 0\n"))
	if err != nil || w.DiscussCap != 60 {
		t.Errorf("default cap: %+v err=%v", w, err)
	}
	for name, body := range map[string]string{
		"bad sum":     "practical: 0.5\nlifestyle: 0.5\ninterpersonal: 0.5\n",
		"unknown key": "practical: 1\nlifestyle: 0\ninterpersonal: 0\nvibes: 9\n",
		"missing key": "practical: 1\nlifestyle: 0\n",
		"bad number":  "practical: lots\nlifestyle: 0\ninterpersonal: 1\n",
		"bad shape":   "practical 0.5\nlifestyle: 0.5\ninterpersonal: 0\n",
	} {
		if _, err := LoadWeights(write(t, body)); err == nil {
			t.Errorf("%s: want error", name)
		}
	}
}
