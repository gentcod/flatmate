package web

import (
	"database/sql"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gentcod/flatmate/internal/ai"
	"github.com/gentcod/flatmate/internal/database"
	"github.com/gentcod/flatmate/internal/match"
	"github.com/gentcod/flatmate/internal/profile"
)

// expenseView is one shared expense from the viewer's side.
type expenseView struct {
	ID        int64
	PaidByMe  bool
	Amount    int64
	Currency  string
	Note      string
	Split     string
	OwedByMe  int64
	OwedToMe  int64
	CreatedAt string
}

// balanceView is a per-currency running total. Positive means the
// peer owes me.
type balanceView struct {
	Currency string
	Amount   int64
}

// Abs renders the amount without sign for "you owe" lines.
func (b balanceView) Abs() int64 {
	if b.Amount < 0 {
		return -b.Amount
	}
	return b.Amount
}

// ledger loads the shared expenses between me and peer with balances.
func (s *Server) ledger(r *http.Request, me, peer int64) ([]expenseView, []balanceView, error) {
	rows, err := database.New(s.DB).ListExpenses(r.Context(), database.ListExpensesParams{
		S1: me, R1: peer, S2: peer, R2: me,
	})
	if err != nil {
		return nil, nil, err
	}
	out := []expenseView{}
	totals := map[string]int64{}
	for _, e := range rows {
		v := expenseView{ID: e.ID, PaidByMe: e.Payer == me, Amount: e.Amount,
			Currency: e.Currency, Note: e.Note, Split: e.Split, CreatedAt: e.CreatedAt}
		if e.Payer == me {
			v.OwedToMe = e.PeerShare
			totals[e.Currency] += e.PeerShare
		} else {
			v.OwedByMe = e.PeerShare
			totals[e.Currency] -= e.PeerShare
		}
		out = append(out, v)
	}
	bals := []balanceView{}
	for cur, amt := range totals {
		bals = append(bals, balanceView{Currency: cur, Amount: amt})
	}
	sort.Slice(bals, func(i, j int) bool { return bals[i].Currency < bals[j].Currency })
	return out, bals, nil
}

func checkExpense(amount int64, currency, note, split string, payerShare, peerShare int64) error {
	switch split {
	case "equal", "mine", "theirs", "custom":
	default:
		return fmt.Errorf("unknown split, want equal|mine|theirs|custom")
	}
	switch {
	case amount <= 0 || amount > 1000000000000:
		return fmt.Errorf("amount must be between 1 and 1000000000000")
	case len(currency) == 0 || len(currency) > 8:
		return fmt.Errorf("currency is invalid")
	case len(note) > 200:
		return fmt.Errorf("note must be at most 200 characters")
	case payerShare < 0 || peerShare < 0 || payerShare+peerShare != amount:
		return fmt.Errorf("shares must be non-negative and sum to the amount")
	}
	return nil
}

// splitShares resolves preset labels into concrete shares. "mine" and
// "theirs" are relative to the payer.
func splitShares(split string, amount, payerShare, peerShare int64) (int64, int64) {
	switch split {
	case "equal":
		half := amount / 2
		return amount - half, half
	case "mine":
		return amount, 0
	case "theirs":
		return 0, amount
	}
	return payerShare, peerShare
}

func checkSearch(name, area string, maxRent int64) error {
	switch {
	case len(name) == 0 || len(name) > 40:
		return fmt.Errorf("name must be 1–40 characters")
	case len(area) > 120:
		return fmt.Errorf("area must be at most 120 characters")
	case maxRent < 0:
		return fmt.Errorf("max rent cannot be negative")
	}
	return nil
}

type apiExpense struct {
	ID        int64  `json:"id"`
	PaidByMe  bool   `json:"paid_by_me"`
	Amount    int64  `json:"amount"`
	Currency  string `json:"currency"`
	Note      string `json:"note"`
	Split     string `json:"split"`
	OwedByMe  int64  `json:"owed_by_me"`
	OwedToMe  int64  `json:"owed_to_me"`
	CreatedAt string `json:"created_at"`
}

type apiBalance struct {
	Currency string `json:"currency"`
	Amount   int64  `json:"amount"`
}

func (s *Server) apiExpensesGet(w http.ResponseWriter, r *http.Request) {
	id := authedID(r)
	peer, p, ok := s.apiThreadPeer(w, r, id)
	if !ok {
		return
	}
	exp, bals, err := s.ledger(r, id, peer)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Load error.")
		return
	}
	out := []apiExpense{}
	for _, e := range exp {
		out = append(out, apiExpense{ID: e.ID, PaidByMe: e.PaidByMe, Amount: e.Amount,
			Currency: e.Currency, Note: e.Note, Split: e.Split,
			OwedByMe: e.OwedByMe, OwedToMe: e.OwedToMe, CreatedAt: e.CreatedAt})
	}
	bout := []apiBalance{}
	for _, b := range bals {
		bout = append(bout, apiBalance{Currency: b.Currency, Amount: b.Amount})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"peer":     map[string]any{"id": peer, "first_name": p.FirstName},
		"expenses": out, "balances": bout,
	})
}

func (s *Server) apiExpensesPost(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRFAPI(w, r) {
		return
	}
	id := authedID(r)
	peer, ok := pathID(r)
	if !ok || peer == id {
		writeErr(w, http.StatusNotFound, "Not found.")
		return
	}
	if code, msg := s.messageGate(r, id, peer); code != 0 {
		writeErr(w, code, msg)
		return
	}
	var in struct {
		Amount     int64  `json:"amount"`
		Currency   string `json:"currency"`
		Note       string `json:"note"`
		Split      string `json:"split"`
		Payer      string `json:"payer"` // "me" (default) or "them"
		PayerShare int64  `json:"payer_share"`
		PeerShare  int64  `json:"peer_share"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	currency := strings.ToUpper(strings.TrimSpace(in.Currency))
	if currency == "" {
		currency = "USD"
	}
	note := strings.TrimSpace(in.Note)
	split := strings.TrimSpace(in.Split)
	if split == "" {
		split = "equal"
	}
	payer, other := id, peer
	if in.Payer == "them" {
		payer, other = peer, id
	}
	ps, qs := splitShares(split, in.Amount, in.PayerShare, in.PeerShare)
	if err := checkExpense(in.Amount, currency, note, split, ps, qs); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if !s.MsgLimiter.Allow("exp:user:" + strconv.FormatInt(id, 10)) {
		writeErr(w, http.StatusTooManyRequests, "Slow down a touch and try again.")
		return
	}
	if err := database.New(s.DB).InsertExpense(r.Context(), database.InsertExpenseParams{
		Payer: payer, Peer: other, Amount: in.Amount, Currency: currency,
		Note: note, Split: split, PayerShare: ps, PeerShare: qs, CreatedAt: nowRFC3339(),
	}); err != nil {
		writeErr(w, http.StatusInternalServerError, "Something went wrong. Try again.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]bool{"ok": true})
}

func (s *Server) apiExpenseDelete(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRFAPI(w, r) {
		return
	}
	id := authedID(r)
	target, ok := pathID(r)
	if !ok {
		writeErr(w, http.StatusNotFound, "Not found.")
		return
	}
	e, err := database.New(s.DB).GetExpense(r.Context(), target)
	if err == sql.ErrNoRows {
		writeErr(w, http.StatusNotFound, "Not found.")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Load error.")
		return
	}
	if e.Payer != id && e.Peer != id {
		writeErr(w, http.StatusNotFound, "Not found.")
		return
	}
	other := e.Payer
	if other == id {
		other = e.Peer
	}
	if s.blockedPair(r, id, other) || s.deactivated(r, other) {
		writeErr(w, http.StatusNotFound, "Not found.")
		return
	}
	if err := database.New(s.DB).DeleteExpense(r.Context(), e.ID); err != nil {
		writeErr(w, http.StatusInternalServerError, "Something went wrong. Try again.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

type apiSearch struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Area        string `json:"area"`
	MaxRent     int64  `json:"max_rent"`
	CreatedAt   string `json:"created_at"`
	NewListings int64  `json:"new_listings"`
}

// searchCounts lists the caller's saved searches with the number of
// matching listings posted since each search was saved. Timestamps
// are RFC3339, so string comparison orders them.
func (s *Server) searchCounts(r *http.Request, id int64) ([]apiSearch, error) {
	searches, err := database.New(s.DB).ListSearches(r.Context(), id)
	if err != nil {
		return nil, err
	}
	rows, err := database.New(s.DB).ListListings(r.Context(), 200)
	if err != nil {
		return nil, err
	}
	out := []apiSearch{}
	for _, sc := range searches {
		var n int64
		area := strings.ToLower(sc.Area)
		for _, l := range rows {
			if l.CreatedAt <= sc.CreatedAt {
				continue
			}
			if area != "" && !strings.Contains(strings.ToLower(l.AreaText), area) {
				continue
			}
			if sc.MaxRent > 0 && l.MonthlyRent > sc.MaxRent {
				continue
			}
			if !s.listingVisible(r, id, l) {
				continue
			}
			n++
		}
		out = append(out, apiSearch{ID: sc.ID, Name: sc.Name, Area: sc.Area,
			MaxRent: sc.MaxRent, CreatedAt: sc.CreatedAt, NewListings: n})
	}
	return out, nil
}

func (s *Server) apiSearchesList(w http.ResponseWriter, r *http.Request) {
	id := authedID(r)
	out, err := s.searchCounts(r, id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Load error.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"searches": out})
}

func (s *Server) apiSearchesCreate(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRFAPI(w, r) {
		return
	}
	id := authedID(r)
	var in struct {
		Name    string `json:"name"`
		Area    string `json:"area"`
		MaxRent int64  `json:"max_rent"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	name := strings.TrimSpace(in.Name)
	area := strings.TrimSpace(in.Area)
	if err := checkSearch(name, area, in.MaxRent); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	res, err := database.New(s.DB).CreateSearch(r.Context(), database.CreateSearchParams{
		UserID: id, Name: name, Area: area, MaxRent: in.MaxRent, CreatedAt: nowRFC3339(),
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Something went wrong. Try again.")
		return
	}
	newID, err := res.LastInsertId()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Something went wrong. Try again.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": newID})
}

func (s *Server) apiSearchesDelete(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRFAPI(w, r) {
		return
	}
	id := authedID(r)
	target, ok := pathID(r)
	if !ok {
		writeErr(w, http.StatusNotFound, "Not found.")
		return
	}
	// Scoped to the owner; deleting someone else's id is a silent no-op.
	database.New(s.DB).DeleteSearch(r.Context(), database.DeleteSearchParams{ID: target, UserID: id})
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

type apiAgreementSection struct {
	Heading string   `json:"heading"`
	Points  []string `json:"points"`
}

func (s *Server) apiAgreementPost(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRFAPI(w, r) {
		return
	}
	id := authedID(r)
	target, ok := pathID(r)
	if !ok || target == id {
		writeErr(w, http.StatusNotFound, "Not found.")
		return
	}
	tp, found, err := profile.Get(r.Context(), s.DB, target)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Load error.")
		return
	}
	if !found || !tp.Searching || s.blockedPair(r, id, target) || s.deactivated(r, target) {
		writeErr(w, http.StatusNotFound, "Not found.")
		return
	}
	var flagged []string
	if me, ok := s.tryCandidate(r, id); ok {
		if them, ok := s.tryCandidate(r, target); ok {
			bd := match.Score(me, them, s.Weights)
			for _, dr := range bd.Dimensions {
				if dr.Discuss && dr.Name != "budget" && dr.Name != "area" && dr.Name != "dates" {
					flagged = append(flagged, dr.Name)
				}
			}
		}
	}
	sections := ai.TemplateAgreement()
	enhanced := false
	if s.AILimiter.Allow("user:" + strconv.FormatInt(id, 10)) {
		if draft, ok := ai.DraftAgreement(r.Context(), s.AI, s.authorName(r, id), tp.FirstName, flagged); ok {
			sections, enhanced = draft, true
		}
	}
	out := []apiAgreementSection{}
	for _, sec := range sections {
		out = append(out, apiAgreementSection{Heading: sec.Heading, Points: sec.Points})
	}
	writeJSON(w, http.StatusOK, map[string]any{"sections": out, "enhanced": enhanced})
}

func (s *Server) apiMatchesSearch(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRFAPI(w, r) {
		return
	}
	id := authedID(r)
	me, ok := s.apiCandidate(w, r, id)
	if !ok {
		return
	}
	var in struct {
		Query string `json:"query"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	q := strings.TrimSpace(in.Query)
	if len(q) > 200 {
		q = q[:200]
	}
	filters, enhanced := ai.KeywordSearch(q), false
	if q != "" && s.AILimiter.Allow("user:"+strconv.FormatInt(id, 10)) {
		if f, ok := ai.ParseSearch(r.Context(), s.AI, q); ok {
			filters, enhanced = f, true
		}
	}
	others := s.loadCandidates(r, id)
	blocked := s.blockSet(r, id)
	acts := s.actionSet(r, id)
	toMe := s.interestToMe(r, id)
	reported := s.reportedByMe(r, id)
	views := []apiMatch{}
	for _, c := range others {
		if blocked[c.id] || reported[c.id] || acts[c.id]["dismiss"] {
			continue
		}
		if !nlKeep(filters, c.p, c.a) {
			continue
		}
		cand := toMatchCandidate(c.p, c.a)
		if !match.Eligible(me, cand, false, false) {
			continue
		}
		bd := match.Score(me, cand, s.Weights)
		top := ""
		if len(bd.Alignments) > 0 {
			top = bd.Alignments[0]
		}
		views = append(views, apiMatch{
			ID: c.id, FirstName: c.p.FirstName, AreaText: c.p.AreaText,
			Gender: profile.GenderLabel(c.p.Gender), Occupation: c.p.Occupation,
			Bio: c.p.Bio, Total: bd.Total, TopAlign: top,
			Saved: acts[c.id]["save"], Mutual: acts[c.id]["interest"] && toMe[c.id],
		})
	}
	sort.Slice(views, func(i, j int) bool { return views[i].Total > views[j].Total })
	writeJSON(w, http.StatusOK, map[string]any{
		"matches": views, "empty": len(views) == 0, "enhanced": enhanced,
		"description": nlDescribe(filters),
		"filters": map[string]any{"area": filters.Area, "max_rent": filters.MaxRent,
			"smoking": filters.Smoking, "pets": filters.Pets, "wfh_max": filters.WFHMax},
	})
}
