package web

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"github.com/gentcod/flatmate/internal/database"
	"github.com/gentcod/flatmate/internal/profile"
)

type apiListing struct {
	ID          int64  `json:"id"`
	UserID      int64  `json:"user_id"`
	OwnerName   string `json:"owner_name"`
	Title       string `json:"title"`
	Description string `json:"description"`
	AreaText    string `json:"area_text"`
	MonthlyRent int64  `json:"monthly_rent"`
	Currency    string `json:"currency"`
	PhotoURL    string `json:"photo_url"`
	CreatedAt   string `json:"created_at"`
	Status      string `json:"status"`
	Mine        bool   `json:"mine"`
}

func (s *Server) toAPIListing(r *http.Request, id int64, l database.Listing) apiListing {
	return apiListing{
		ID: l.ID, UserID: l.UserID, OwnerName: s.authorName(r, l.UserID),
		Title: l.Title, Description: l.Description, AreaText: l.AreaText,
		MonthlyRent: l.MonthlyRent, Currency: l.Currency, PhotoURL: l.PhotoUrl, CreatedAt: l.CreatedAt,
		Status: l.Status, Mine: l.UserID == id,
	}
}

func (s *Server) apiListingsList(w http.ResponseWriter, r *http.Request) {
	id := authedID(r)
	var rows []database.Listing
	var err error
	if r.URL.Query().Get("mine") == "1" {
		rows, err = database.New(s.DB).ListListingsByUser(r.Context(), id)
	} else {
		rows, err = database.New(s.DB).ListListings(r.Context(), 200)
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Load error.")
		return
	}
	area := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("area")))
	var maxRent int64
	if q := strings.TrimSpace(r.URL.Query().Get("max_rent")); q != "" {
		maxRent, _ = strconv.ParseInt(q, 10, 64)
	}
	out := []apiListing{}
	for _, l := range rows {
		if !s.listingVisible(r, id, l) {
			continue
		}
		if area != "" && !strings.Contains(strings.ToLower(l.AreaText), area) {
			continue
		}
		if maxRent > 0 && l.MonthlyRent > maxRent {
			continue
		}
		out = append(out, s.toAPIListing(r, id, l))
	}
	writeJSON(w, http.StatusOK, map[string]any{"listings": out})
}

func (s *Server) apiListingsCreate(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRFAPI(w, r) {
		return
	}
	id := authedID(r)
	if _, found, err := profile.Get(r.Context(), s.DB, id); err != nil || !found {
		writeJSON(w, http.StatusConflict, map[string]string{
			"error": "Complete your profile first.", "setup_required": "profile"})
		return
	}
	var in struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		AreaText    string `json:"area_text"`
		MonthlyRent int64  `json:"monthly_rent"`
		Currency    string `json:"currency"`
		PhotoURL    string `json:"photo_url"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	title := strings.TrimSpace(in.Title)
	description := strings.TrimSpace(in.Description)
	area := strings.TrimSpace(in.AreaText)
	currency := strings.ToUpper(strings.TrimSpace(in.Currency))
	if currency == "" {
		currency = "USD"
	}
	photo := strings.TrimSpace(in.PhotoURL)
	if err := checkListing(title, description, area, currency, in.MonthlyRent); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := checkPhotoURL(photo); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	now := nowRFC3339()
	res, err := database.New(s.DB).CreateListing(r.Context(), database.CreateListingParams{
		UserID: id, Title: title, Description: description, AreaText: area,
		MonthlyRent: in.MonthlyRent, Currency: currency, PhotoUrl: photo, CreatedAt: now, UpdatedAt: now,
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

// apiListingOwned loads a listing the caller may see; ownerOnly
// restricts to the owner. Failures are always 404 so listings never
// leak across owners.
func (s *Server) apiListingOwned(w http.ResponseWriter, r *http.Request, id int64, ownerOnly bool) (database.Listing, bool) {
	target, ok := pathID(r)
	if !ok {
		writeErr(w, http.StatusNotFound, "Not found.")
		return database.Listing{}, false
	}
	l, err := database.New(s.DB).GetListing(r.Context(), target)
	if err == sql.ErrNoRows {
		writeErr(w, http.StatusNotFound, "Not found.")
		return database.Listing{}, false
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Load error.")
		return database.Listing{}, false
	}
	if ownerOnly {
		if l.UserID != id {
			writeErr(w, http.StatusNotFound, "Not found.")
			return database.Listing{}, false
		}
	} else if !s.listingVisible(r, id, l) {
		writeErr(w, http.StatusNotFound, "Not found.")
		return database.Listing{}, false
	}
	return l, true
}

func (s *Server) apiListingGet(w http.ResponseWriter, r *http.Request) {
	id := authedID(r)
	l, ok := s.apiListingOwned(w, r, id, false)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"listing": s.toAPIListing(r, id, l)})
}

func (s *Server) apiListingPut(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRFAPI(w, r) {
		return
	}
	id := authedID(r)
	l, ok := s.apiListingOwned(w, r, id, true)
	if !ok {
		return
	}
	var in struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		AreaText    string `json:"area_text"`
		MonthlyRent int64  `json:"monthly_rent"`
		Currency    string `json:"currency"`
		PhotoURL    string `json:"photo_url"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	title := strings.TrimSpace(in.Title)
	description := strings.TrimSpace(in.Description)
	area := strings.TrimSpace(in.AreaText)
	currency := strings.ToUpper(strings.TrimSpace(in.Currency))
	if currency == "" {
		currency = "USD"
	}
	photo := strings.TrimSpace(in.PhotoURL)
	if err := checkListing(title, description, area, currency, in.MonthlyRent); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := checkPhotoURL(photo); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	err := database.New(s.DB).UpdateListing(r.Context(), database.UpdateListingParams{
		Title: title, Description: description, AreaText: area,
		MonthlyRent: in.MonthlyRent, Currency: currency, PhotoUrl: photo, UpdatedAt: nowRFC3339(),
		ID: l.ID, UserID: id,
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Something went wrong. Try again.")
		return
	}
	updated, err := database.New(s.DB).GetListing(r.Context(), l.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Load error.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"listing": s.toAPIListing(r, id, updated)})
}

func (s *Server) apiListingDelete(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRFAPI(w, r) {
		return
	}
	id := authedID(r)
	l, ok := s.apiListingOwned(w, r, id, true)
	if !ok {
		return
	}
	if err := database.New(s.DB).DeleteListing(r.Context(), database.DeleteListingParams{ID: l.ID, UserID: id}); err != nil {
		writeErr(w, http.StatusInternalServerError, "Something went wrong. Try again.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

type apiConversation struct {
	PeerID      int64  `json:"peer_id"`
	PeerName    string `json:"peer_name"`
	LastBody    string `json:"last_body"`
	LastAt      string `json:"last_at"`
	LastFromMe  bool   `json:"last_from_me"`
	UnreadCount int64  `json:"unread_count"`
}

func (s *Server) apiConversationsList(w http.ResponseWriter, r *http.Request) {
	id := authedID(r)
	items, err := s.inbox(r, id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Load error.")
		return
	}
	out := []apiConversation{}
	for _, it := range items {
		out = append(out, apiConversation{
			PeerID: it.PeerID, PeerName: it.PeerName,
			LastBody: it.LastBody, LastAt: it.LastAt, LastFromMe: it.LastFromMe,
			UnreadCount: it.Unread,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"conversations": out})
}

type apiMessage struct {
	ID        int64  `json:"id"`
	FromMe    bool   `json:"from_me"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
}

// apiThreadPeer resolves the peer for a thread view. Reading history
// stays possible after hiding or unmatching; only blocks and
// deactivation close it.
func (s *Server) apiThreadPeer(w http.ResponseWriter, r *http.Request, id int64) (int64, *profile.Profile, bool) {
	peer, ok := pathID(r)
	if !ok || peer == id {
		writeErr(w, http.StatusNotFound, "Not found.")
		return 0, nil, false
	}
	if s.blockedPair(r, id, peer) || s.deactivated(r, peer) {
		writeErr(w, http.StatusNotFound, "Not found.")
		return 0, nil, false
	}
	p, found, err := profile.Get(r.Context(), s.DB, peer)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Load error.")
		return 0, nil, false
	}
	if !found {
		writeErr(w, http.StatusNotFound, "Not found.")
		return 0, nil, false
	}
	return peer, p, true
}

func (s *Server) apiThreadGet(w http.ResponseWriter, r *http.Request) {
	id := authedID(r)
	peer, p, ok := s.apiThreadPeer(w, r, id)
	if !ok {
		return
	}
	rows, err := database.New(s.DB).ListConversation(r.Context(), database.ListConversationParams{
		S1: id, R1: peer, S2: peer, R2: id, Limit: 100,
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Load error.")
		return
	}
	_ = database.New(s.DB).MarkThreadRead(r.Context(), database.MarkThreadReadParams{
		ReadAt: sql.NullString{String: nowRFC3339(), Valid: true}, Sender: peer, Recipient: id,
	})
	out := []apiMessage{}
	for _, m := range rows {
		out = append(out, apiMessage{ID: m.ID, FromMe: m.Sender == id, Body: m.Body, CreatedAt: m.CreatedAt})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"peer":     map[string]any{"id": peer, "first_name": p.FirstName},
		"messages": out,
	})
}

func (s *Server) apiThreadPost(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRFAPI(w, r) {
		return
	}
	id := authedID(r)
	peer, ok := pathID(r)
	if !ok {
		writeErr(w, http.StatusNotFound, "Not found.")
		return
	}
	if code, msg := s.messageGate(r, id, peer); code != 0 {
		writeErr(w, code, msg)
		return
	}
	var in struct {
		Body string `json:"body"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	body := strings.TrimSpace(in.Body)
	if err := checkMessage(body); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if !s.MsgLimiter.Allow("user:" + strconv.FormatInt(id, 10)) {
		writeErr(w, http.StatusTooManyRequests, "Slow down a touch and try again.")
		return
	}
	if err := database.New(s.DB).InsertMessage(r.Context(), database.InsertMessageParams{
		Sender: id, Recipient: peer, Body: body, CreatedAt: nowRFC3339(),
	}); err != nil {
		writeErr(w, http.StatusInternalServerError, "Something went wrong. Try again.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]bool{"ok": true})
}

type apiReview struct {
	AuthorID     int64  `json:"author_id"`
	AuthorName   string `json:"author_name"`
	Rating       int64  `json:"rating"`
	Body         string `json:"body"`
	CreatedAt    string `json:"created_at"`
	VerifiedStay bool   `json:"verified_stay"`
}

// apiReviewTarget resolves the user being reviewed. Anyone with an
// account can receive (and be read about); only blocks and
// deactivation hide them.
func (s *Server) apiReviewTarget(w http.ResponseWriter, r *http.Request, id int64) (int64, bool) {
	target, ok := pathID(r)
	if !ok || target == id {
		writeErr(w, http.StatusNotFound, "Not found.")
		return 0, false
	}
	if s.blockedPair(r, id, target) || s.deactivated(r, target) {
		writeErr(w, http.StatusNotFound, "Not found.")
		return 0, false
	}
	return target, true
}

func (s *Server) toAPIReviews(r *http.Request, target int64) ([]apiReview, error) {
	rows, err := database.New(s.DB).ListReviewsForUser(r.Context(), target)
	if err != nil {
		return nil, err
	}
	out := []apiReview{}
	for _, rv := range rows {
		out = append(out, apiReview{
			AuthorID: rv.Author, AuthorName: s.authorName(r, rv.Author),
			Rating: rv.Rating, Body: rv.Body, CreatedAt: rv.CreatedAt,
			VerifiedStay: s.verifiedStay(r, rv.Author, target),
		})
	}
	return out, nil
}

func (s *Server) apiReviewsGet(w http.ResponseWriter, r *http.Request) {
	id := authedID(r)
	target, ok := s.apiReviewTarget(w, r, id)
	if !ok {
		return
	}
	out, err := s.toAPIReviews(r, target)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Load error.")
		return
	}
	var sum int64
	for _, rv := range out {
		sum += rv.Rating
	}
	avg := 0.0
	if len(out) > 0 {
		avg = float64(sum) / float64(len(out))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"reviews": out, "avg": avg, "count": len(out),
	})
}

func (s *Server) apiReviewsPut(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRFAPI(w, r) {
		return
	}
	id := authedID(r)
	target, ok := s.apiReviewTarget(w, r, id)
	if !ok {
		return
	}
	var in struct {
		Rating int64  `json:"rating"`
		Body   string `json:"body"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	body := strings.TrimSpace(in.Body)
	if err := checkReview(in.Rating, body); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if !s.verifiedStay(r, id, target) {
		writeErr(w, http.StatusForbidden, "You can review after a verified stay together.")
		return
	}
	if err := database.New(s.DB).UpsertReview(r.Context(), database.UpsertReviewParams{
		Author: id, Target: target, Rating: in.Rating, Body: body, CreatedAt: nowRFC3339(),
	}); err != nil {
		writeErr(w, http.StatusInternalServerError, "Something went wrong. Try again.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

type apiStay struct {
	ID           int64  `json:"id"`
	PeerID       int64  `json:"peer_id"`
	PeerName     string `json:"peer_name"`
	Status       string `json:"status"`
	ProposedByMe bool   `json:"proposed_by_me"`
	ListingID    int64  `json:"listing_id"`
	ListingTitle string `json:"listing_title"`
}

func (s *Server) toAPIStay(r *http.Request, id int64, st database.Stay) apiStay {
	peer := st.UserA
	if peer == id {
		peer = st.UserB
	}
	out := apiStay{
		ID: st.ID, PeerID: peer, PeerName: s.authorName(r, peer),
		Status: st.Status, ProposedByMe: st.ProposedBy == id,
	}
	if st.ListingID.Valid {
		out.ListingID = st.ListingID.Int64
		if l, err := database.New(s.DB).GetListing(r.Context(), st.ListingID.Int64); err == nil {
			out.ListingTitle = l.Title
		}
	}
	return out
}

func (s *Server) apiStaysList(w http.ResponseWriter, r *http.Request) {
	id := authedID(r)
	rows, err := database.New(s.DB).ListStaysForUser(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Load error.")
		return
	}
	out := []apiStay{}
	for _, st := range rows {
		out = append(out, s.toAPIStay(r, id, st))
	}
	writeJSON(w, http.StatusOK, map[string]any{"stays": out})
}

func (s *Server) apiStaysPropose(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRFAPI(w, r) {
		return
	}
	id := authedID(r)
	var in struct {
		PeerID    int64 `json:"peer_id"`
		ListingID int64 `json:"listing_id"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if in.PeerID == id {
		writeErr(w, http.StatusNotFound, "Not found.")
		return
	}
	if code, msg := s.messageGate(r, id, in.PeerID); code != 0 {
		writeErr(w, code, msg)
		return
	}
	var listing sql.NullInt64
	if in.ListingID > 0 {
		l, err := database.New(s.DB).GetListing(r.Context(), in.ListingID)
		if err == sql.ErrNoRows {
			writeErr(w, http.StatusNotFound, "Not found.")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "Load error.")
			return
		}
		if l.UserID != id && l.UserID != in.PeerID {
			writeErr(w, http.StatusBadRequest, "That room belongs to neither of you.")
			return
		}
		listing = sql.NullInt64{Int64: in.ListingID, Valid: true}
	}
	a, b := canonPair(id, in.PeerID)
	now := nowRFC3339()
	if st, err := database.New(s.DB).GetStayBetween(r.Context(), database.GetStayBetweenParams{A: a, B: b}); err == nil {
		switch st.Status {
		case StayDeclined:
			if err := database.New(s.DB).UpdateStayProposal(r.Context(), database.UpdateStayProposalParams{
				ProposedBy: id, ListingID: listing, Status: StayProposed, UpdatedAt: now, ID: st.ID,
			}); err != nil {
				writeErr(w, http.StatusInternalServerError, "Something went wrong. Try again.")
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"id": st.ID, "status": StayProposed})
		case StayEnded:
			writeErr(w, http.StatusConflict, "You already have stay history together.")
		default:
			writeErr(w, http.StatusConflict, "A stay is already in progress.")
		}
		return
	} else if err != sql.ErrNoRows {
		writeErr(w, http.StatusInternalServerError, "Load error.")
		return
	}
	res, err := database.New(s.DB).CreateStay(r.Context(), database.CreateStayParams{
		UserA: a, UserB: b, ProposedBy: id, ListingID: listing,
		Status: StayProposed, CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Something went wrong. Try again.")
		return
	}
	stayID, _ := res.LastInsertId()
	writeJSON(w, http.StatusCreated, map[string]any{"id": stayID, "status": StayProposed})
}

func (s *Server) apiStayTransition(w http.ResponseWriter, r *http.Request, action string) {
	if !s.checkCSRFAPI(w, r) {
		return
	}
	id := authedID(r)
	stayID, ok := pathID(r)
	if !ok {
		writeErr(w, http.StatusNotFound, "Not found.")
		return
	}
	st, err := database.New(s.DB).GetStay(r.Context(), stayID)
	if err == sql.ErrNoRows || (st.UserA != id && st.UserB != id) {
		writeErr(w, http.StatusNotFound, "Not found.")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Load error.")
		return
	}
	next := ""
	switch action {
	case "accept", "decline":
		if st.Status != StayProposed || st.ProposedBy == id {
			writeErr(w, http.StatusConflict, "Nothing to respond to.")
			return
		}
		if action == "accept" {
			next = StayActive
		} else {
			next = StayDeclined
		}
	case "end":
		if st.Status != StayActive {
			writeErr(w, http.StatusConflict, "No active stay to end.")
			return
		}
		next = StayEnded
	}
	if err := database.New(s.DB).UpdateStayStatus(r.Context(), database.UpdateStayStatusParams{
		Status: next, UpdatedAt: nowRFC3339(), ID: st.ID,
	}); err != nil {
		writeErr(w, http.StatusInternalServerError, "Something went wrong. Try again.")
		return
	}
	if next == StayActive && st.ListingID.Valid {
		_ = database.New(s.DB).UpdateListingStatus(r.Context(), database.UpdateListingStatusParams{
			Status: "rented", UpdatedAt: nowRFC3339(), ID: st.ListingID.Int64,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": st.ID, "status": next})
}

func (s *Server) apiStayAccept(w http.ResponseWriter, r *http.Request) {
	s.apiStayTransition(w, r, "accept")
}

func (s *Server) apiStayDecline(w http.ResponseWriter, r *http.Request) {
	s.apiStayTransition(w, r, "decline")
}

func (s *Server) apiStayEnd(w http.ResponseWriter, r *http.Request) {
	s.apiStayTransition(w, r, "end")
}

type apiListingReview struct {
	AuthorID   int64  `json:"author_id"`
	AuthorName string `json:"author_name"`
	Rating     int64  `json:"rating"`
	Body       string `json:"body"`
	Kind       string `json:"kind"`
	CreatedAt  string `json:"created_at"`
}

func (s *Server) apiListingReviewsGet(w http.ResponseWriter, r *http.Request) {
	id := authedID(r)
	l, ok := s.apiListingOwned(w, r, id, false)
	if !ok {
		return
	}
	rows, err := database.New(s.DB).ListListingReviews(r.Context(), l.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Load error.")
		return
	}
	stays, visits := []apiListingReview{}, []apiListingReview{}
	var ratings, weights []int64
	for _, rv := range rows {
		out := apiListingReview{
			AuthorID: rv.Author, AuthorName: s.authorName(r, rv.Author),
			Rating: rv.Rating, Body: rv.Body, Kind: rv.Kind, CreatedAt: rv.CreatedAt,
		}
		w := int64(1)
		if rv.Kind == "stay" {
			stays = append(stays, out)
			w = 3
		} else {
			visits = append(visits, out)
		}
		ratings = append(ratings, rv.Rating)
		weights = append(weights, w)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"stay_reviews": stays, "visit_reviews": visits,
		"trust_score": trustScore(ratings, weights),
		"stay_count":  len(stays), "visit_count": len(visits),
	})
}

func (s *Server) apiListingReviewsPut(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRFAPI(w, r) {
		return
	}
	id := authedID(r)
	l, ok := s.apiListingOwned(w, r, id, false)
	if !ok {
		return
	}
	if l.UserID == id {
		writeErr(w, http.StatusForbidden, "You can't review your own listing.")
		return
	}
	var in struct {
		Rating int64  `json:"rating"`
		Body   string `json:"body"`
		Kind   string `json:"kind"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	kind := strings.ToLower(strings.TrimSpace(in.Kind))
	if kind != "stay" && kind != "visit" {
		writeErr(w, http.StatusBadRequest, "Kind must be stay or visit.")
		return
	}
	if kind == "stay" && !s.tenantStay(r, l.ID, id) {
		writeErr(w, http.StatusForbidden, "Only verified tenants can leave a stay review.")
		return
	}
	body := strings.TrimSpace(in.Body)
	if err := checkReview(in.Rating, body); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := database.New(s.DB).UpsertListingReview(r.Context(), database.UpsertListingReviewParams{
		ListingID: l.ID, Author: id, Rating: in.Rating, Body: body, Kind: kind, CreatedAt: nowRFC3339(),
	}); err != nil {
		writeErr(w, http.StatusInternalServerError, "Something went wrong. Try again.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) apiListingStatus(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRFAPI(w, r) {
		return
	}
	id := authedID(r)
	l, ok := s.apiListingOwned(w, r, id, true)
	if !ok {
		return
	}
	var in struct {
		Status string `json:"status"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	status := strings.ToLower(strings.TrimSpace(in.Status))
	if status != "available" && status != "rented" {
		writeErr(w, http.StatusBadRequest, "Status must be available or rented.")
		return
	}
	if err := database.New(s.DB).UpdateListingStatus(r.Context(), database.UpdateListingStatusParams{
		Status: status, UpdatedAt: nowRFC3339(), ID: l.ID,
	}); err != nil {
		writeErr(w, http.StatusInternalServerError, "Something went wrong. Try again.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": l.ID, "status": status})
}
