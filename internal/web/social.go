package web

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gentcod/flatmate/internal/ai"
	"github.com/gentcod/flatmate/internal/database"
	"github.com/gentcod/flatmate/internal/match"
	"github.com/gentcod/flatmate/internal/profile"
)

// mutualInterest reports whether id and target expressed interest both ways.
func (s *Server) mutualInterest(r *http.Request, id, target int64) bool {
	q := database.New(s.DB)
	mine, _ := q.CountInterestFrom(r.Context(), database.CountInterestFromParams{Actor: id, Target: target})
	theirs, _ := q.CountInterestFrom(r.Context(), database.CountInterestFromParams{Actor: target, Target: id})
	return mine > 0 && theirs > 0
}

// Stay statuses for the co-rental handshake.
const (
	StayProposed = "proposed"
	StayActive   = "active"
	StayEnded    = "ended"
	StayDeclined = "declined"
)

// canonPair orders a user pair for stay lookups.
func canonPair(a, b int64) (int64, int64) {
	if a < b {
		return a, b
	}
	return b, a
}

func (s *Server) stayBetween(r *http.Request, id, peer int64) (database.Stay, bool) {
	a, b := canonPair(id, peer)
	st, err := database.New(s.DB).GetStayBetween(r.Context(), database.GetStayBetweenParams{A: a, B: b})
	if err != nil {
		return database.Stay{}, false
	}
	return st, true
}

// verifiedStay reports whether id and peer share an active or ended stay.
func (s *Server) verifiedStay(r *http.Request, id, peer int64) bool {
	st, ok := s.stayBetween(r, id, peer)
	if !ok {
		return false
	}
	return st.Status == StayActive || st.Status == StayEnded
}

// tenantStay reports whether user has an active or ended stay linked to listing.
func (s *Server) tenantStay(r *http.Request, listingID, userID int64) bool {
	st, err := database.New(s.DB).GetStayForListing(r.Context(), database.GetStayForListingParams{ListingID: sql.NullInt64{Int64: listingID, Valid: true}, UserID: userID})
	if err != nil {
		return false
	}
	return st.Status == StayActive || st.Status == StayEnded
}

// notify records a best-effort notification. It never fails the action.
func (s *Server) notify(r *http.Request, userID int64, kind string, actorID, refID int64, body string) {
	if userID == actorID {
		return
	}
	_ = database.New(s.DB).CreateNotification(r.Context(), database.CreateNotificationParams{
		UserID: userID, Kind: kind, ActorID: actorID, RefID: refID, Body: body, CreatedAt: nowRFC3339(),
	})
}

// trustScore computes the weighted mean of ratings: tenant (stay)
// reviews count 3x, visitor reviews 1x. Empty input scores 0.
func trustScore(ratings []int64, weights []int64) float64 {
	var num, den int64
	for i, r := range ratings {
		num += r * weights[i]
		den += weights[i]
	}
	if den == 0 {
		return 0
	}
	return float64(num) / float64(den)
}

// messageGate reports whether id may send messages to target.
// Hidden, blocked, deactivated, and missing targets 404 so chat never
// leaks their existence; non-mutual pairs get an honest 403.
func (s *Server) messageGate(r *http.Request, id, target int64) (code int, msg string) {
	if target == id {
		return http.StatusNotFound, "Not found."
	}
	p, found, err := profile.Get(r.Context(), s.DB, target)
	if err != nil || !found || !p.Searching || s.blockedPair(r, id, target) || s.deactivated(r, target) {
		return http.StatusNotFound, "Not found."
	}
	if !s.mutualInterest(r, id, target) {
		return http.StatusForbidden, "You can message after mutual interest."
	}
	return 0, ""
}

// listingVisible reports whether id may see l. Owners always see their own.
func (s *Server) listingVisible(r *http.Request, id int64, l database.Listing) bool {
	if l.UserID == id {
		return true
	}
	p, found, err := profile.Get(r.Context(), s.DB, l.UserID)
	if err != nil || !found || !p.Searching {
		return false
	}
	return !s.blockedPair(r, id, l.UserID) && !s.deactivated(r, l.UserID)
}

// reviewAvg is the mean rating, or 0 when there are no reviews.
func reviewAvg(rs []database.Review) float64 {
	if len(rs) == 0 {
		return 0
	}
	var sum int64
	for _, rv := range rs {
		sum += rv.Rating
	}
	return float64(sum) / float64(len(rs))
}

// checkListing validates normalized listing fields.
func checkListing(title, description, area, currency string, rent int64) error {
	switch {
	case len(title) == 0 || len(title) > 80:
		return fmt.Errorf("title must be 1–80 characters")
	case len(description) > 1000:
		return fmt.Errorf("description must be at most 1000 characters")
	case len(area) == 0 || len(area) > 120:
		return fmt.Errorf("area must be 1–120 characters")
	case rent < 0:
		return fmt.Errorf("rent cannot be negative")
	case len(currency) == 0 || len(currency) > 8:
		return fmt.Errorf("currency is invalid")
	}
	return nil
}

// checkPhotoURL allows empty (no photo), site-relative paths, or https URLs.
func checkPhotoURL(u string) error {
	if u == "" {
		return nil
	}
	if len(u) > 500 {
		return fmt.Errorf("photo must be at most 500 characters")
	}
	if len(u) > 0 && u[0] == '/' {
		return nil
	}
	if len(u) > 8 && u[:8] == "https://" {
		return nil
	}
	return fmt.Errorf("photo must be a /path or https:// URL")
}

// checkReview validates a normalized review.
func checkReview(rating int64, body string) error {
	switch {
	case rating < 1 || rating > 5:
		return fmt.Errorf("rating must be 1–5")
	case len(body) == 0 || len(body) > 500:
		return fmt.Errorf("review must be 1–500 characters")
	}
	return nil
}

// checkMessage validates a normalized chat message.
func checkMessage(body string) error {
	if len(body) == 0 || len(body) > 2000 {
		return fmt.Errorf("message must be 1–2000 characters")
	}
	return nil
}

// authorName resolves a display name, falling back when the author
// never completed a profile.
func (s *Server) authorName(r *http.Request, id int64) string {
	if p, found, err := profile.Get(r.Context(), s.DB, id); err == nil && found && strings.TrimSpace(p.FirstName) != "" {
		return p.FirstName
	}
	return "A flatmate"
}

// inboxItem is one conversation head for the inbox.
type inboxItem struct {
	PeerID     int64
	PeerName   string
	LastBody   string
	LastAt     string
	LastFromMe bool
	Unread     int64
}

// inbox lists recent conversations, newest first, capped at 50 peers.
func (s *Server) inbox(r *http.Request, id int64) ([]inboxItem, error) {
	rows, err := database.New(s.DB).ListRecentForUser(r.Context(), database.ListRecentForUserParams{A: id, B: id})
	if err != nil {
		return nil, err
	}
	if len(rows) > 500 {
		rows = rows[:500]
	}
	unread := map[int64]int64{}
	if counts, err := database.New(s.DB).UnreadBySender(r.Context(), id); err == nil {
		for _, c := range counts {
			unread[c.Sender] = c.Count
		}
	}
	out := []inboxItem{}
	seen := map[int64]bool{}
	for _, m := range rows {
		if len(out) >= 50 {
			break
		}
		peer := m.Sender
		if peer == id {
			peer = m.Recipient
		}
		if seen[peer] {
			continue
		}
		seen[peer] = true
		out = append(out, inboxItem{
			PeerID: peer, PeerName: s.authorName(r, peer),
			LastBody: m.Body, LastAt: m.CreatedAt, LastFromMe: m.Sender == id,
			Unread: unread[peer],
		})
	}
	return out, nil
}

// nlFilters parses the ?q= natural-language query into match filters.
// AI failure or limiter exhaustion degrades to a keyword area filter.
func (s *Server) nlFilters(r *http.Request, id int64) (ai.SearchFilters, string) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	none := ai.SearchFilters{WFHMax: -1}
	if q == "" {
		return none, ""
	}
	if len(q) > 200 {
		q = q[:200]
	}
	if s.AILimiter.Allow("user:" + strconv.FormatInt(id, 10)) {
		if f, ok := ai.ParseSearch(r.Context(), s.AI, q); ok {
			return f, q
		}
	}
	return ai.KeywordSearch(q), q
}

// nlDescribe renders applied filters for display.
func nlDescribe(f ai.SearchFilters) string {
	parts := []string{}
	if f.Area != "" {
		parts = append(parts, "area: "+f.Area)
	}
	if f.MaxRent > 0 {
		parts = append(parts, "under "+strconv.FormatInt(f.MaxRent, 10))
	}
	if f.Smoking == "no" {
		parts = append(parts, "non-smoking")
	}
	if f.Pets == "none" {
		parts = append(parts, "no pets")
	}
	if f.WFHMax >= 0 {
		parts = append(parts, "wfh ≤ "+strconv.FormatInt(f.WFHMax, 10)+" days")
	}
	return strings.Join(parts, ", ")
}

// nlKeep reports whether a candidate passes NL search filters.
func nlKeep(f ai.SearchFilters, p *profile.Profile, a map[string]profile.Answer) bool {
	if f.Area != "" && !strings.Contains(strings.ToLower(p.AreaText), strings.ToLower(f.Area)) {
		return false
	}
	if f.MaxRent > 0 && p.BudgetMin > f.MaxRent {
		return false
	}
	if f.Smoking != "" {
		ans, ok := a["smoking"]
		if !ok || ans.Value != f.Smoking {
			return false
		}
	}
	if f.Pets != "" {
		ans, ok := a["pets"]
		if !ok || ans.Value != f.Pets {
			return false
		}
	}
	if f.WFHMax >= 0 {
		ans, ok := a["wfh"]
		n, err := strconv.Atoi(ans.Value)
		if !ok || err != nil || int64(n) > f.WFHMax {
			return false
		}
	}
	return true
}

// tryCandidate loads engine input without writing any response,
// for features that degrade instead of gating on setup.
func (s *Server) tryCandidate(r *http.Request, id int64) (match.Candidate, bool) {
	p, found, err := profile.Get(r.Context(), s.DB, id)
	if err != nil || !found {
		return match.Candidate{}, false
	}
	a, err := profile.GetAnswers(r.Context(), s.DB, id)
	if err != nil || len(a) != len(match.Dimensions) {
		return match.Candidate{}, false
	}
	return toMatchCandidate(p, a), true
}
