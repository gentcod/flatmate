package web

import (
	"database/sql"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gentcod/flatmate/internal/database"
	"github.com/gentcod/flatmate/internal/profile"
)

type listingView struct {
	ID          int64
	UserID      int64
	OwnerName   string
	Title       string
	Description string
	AreaText    string
	MonthlyRent int64
	Currency    string
	PhotoURL    string
	Status      string
	Mine        bool
}

type convoView struct {
	PeerID     int64
	PeerName   string
	LastBody   string
	LastAt     string
	LastFromMe bool
	Unread     int64
}

type msgView struct {
	Mine bool
	Body string
	At   string
}

type reviewView struct {
	AuthorName   string
	Rating       int64
	Body         string
	VerifiedStay bool
}

type stayProposalView struct {
	ID       int64
	PeerID   int64
	PeerName string
}

func (s *Server) stayProposals(r *http.Request, id int64) []stayProposalView {
	rows, err := database.New(s.DB).ListStaysForUser(r.Context(), id)
	if err != nil {
		return nil
	}
	out := []stayProposalView{}
	for _, st := range rows {
		if st.Status != StayProposed || st.ProposedBy == id {
			continue
		}
		peer := st.UserA
		if peer == id {
			peer = st.UserB
		}
		out = append(out, stayProposalView{ID: st.ID, PeerID: peer, PeerName: s.authorName(r, peer)})
	}
	return out
}

func (s *Server) toListingViews(r *http.Request, id int64, rows []database.Listing) []listingView {
	out := []listingView{}
	for _, l := range rows {
		out = append(out, listingView{
			ID: l.ID, UserID: l.UserID, OwnerName: s.authorName(r, l.UserID),
			Title: l.Title, Description: l.Description, AreaText: l.AreaText,
			MonthlyRent: l.MonthlyRent, Currency: l.Currency, PhotoURL: l.PhotoUrl, Status: l.Status, Mine: l.UserID == id,
		})
	}
	return out
}

func (s *Server) renderListings(w http.ResponseWriter, r *http.Request, errMsg string) {
	id := authedID(r)
	rows, err := database.New(s.DB).ListListings(r.Context(), 200)
	if err != nil {
		http.Error(w, "load error", http.StatusInternalServerError)
		return
	}
	areaFilter := strings.TrimSpace(r.URL.Query().Get("area"))
	area := strings.ToLower(areaFilter)
	maxRent, _ := parseMoney(r.URL.Query().Get("max_rent"))
	visible := []database.Listing{}
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
		visible = append(visible, l)
	}
	searches, _ := s.searchCounts(r, id)
	maxRentStr := ""
	if maxRent > 0 {
		maxRentStr = strconv.FormatInt(maxRent, 10)
	}
	data := map[string]any{"Title": "Room listings", "Authed": true,
		"Listings": s.toListingViews(r, id, visible), "AreaFilter": areaFilter,
		"MaxRentFilter": maxRentStr, "Searches": searches, "CSRF": s.csrfToken(w, r)}
	if errMsg != "" {
		data["Error"] = errMsg
	}
	s.render(w, r, "listings", data)
}

func (s *Server) handleListingsShow(w http.ResponseWriter, r *http.Request) {
	s.renderListings(w, r, "")
}

func (s *Server) handleListingsPost(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRF(w, r) {
		return
	}
	id := authedID(r)
	if _, found, err := profile.Get(r.Context(), s.DB, id); err != nil || !found {
		http.Redirect(w, r, "/profile", http.StatusSeeOther)
		return
	}
	title := strings.TrimSpace(r.FormValue("title"))
	description := strings.TrimSpace(r.FormValue("description"))
	area := strings.TrimSpace(r.FormValue("area_text"))
	currency := strings.ToUpper(strings.TrimSpace(r.FormValue("currency")))
	if currency == "" {
		currency = "USD"
	}
	rent, err := parseMoney(r.FormValue("monthly_rent"))
	if err != nil {
		s.renderListings(w, r, "Rent must be a plain number, 0 or more.")
		return
	}
	photo := strings.TrimSpace(r.FormValue("photo_url"))
	if err := checkListing(title, description, area, currency, rent); err != nil {
		s.renderListings(w, r, err.Error())
		return
	}
	if err := checkPhotoURL(photo); err != nil {
		s.renderListings(w, r, err.Error())
		return
	}
	now := nowRFC3339()
	if _, err := database.New(s.DB).CreateListing(r.Context(), database.CreateListingParams{
		UserID: id, Title: title, Description: description, AreaText: area,
		MonthlyRent: rent, Currency: currency, PhotoUrl: photo, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		s.renderListings(w, r, "Something went wrong. Try again.")
		return
	}
	http.Redirect(w, r, "/listings", http.StatusSeeOther)
}

// listingOwned loads a listing the caller may see; ownerOnly restricts
// to the owner. Failures always 404 so listings never leak.
func (s *Server) listingOwned(w http.ResponseWriter, r *http.Request, id int64, ownerOnly bool) (database.Listing, bool) {
	target, ok := pathID(r)
	if !ok {
		http.NotFound(w, r)
		return database.Listing{}, false
	}
	l, err := database.New(s.DB).GetListing(r.Context(), target)
	if err != nil {
		http.NotFound(w, r)
		return database.Listing{}, false
	}
	if ownerOnly {
		if l.UserID != id {
			http.NotFound(w, r)
			return database.Listing{}, false
		}
	} else if !s.listingVisible(r, id, l) {
		http.NotFound(w, r)
		return database.Listing{}, false
	}
	return l, true
}

func (s *Server) renderListing(w http.ResponseWriter, r *http.Request, l database.Listing, errMsg string) {
	id := authedID(r)
	views := s.toListingViews(r, id, []database.Listing{l})
	mutual, myInterest, contact, contactSelf := false, false, "", false
	if l.UserID != id {
		mutual = s.mutualInterest(r, id, l.UserID)
		if n, _ := database.New(s.DB).CountInterestFrom(r.Context(), database.CountInterestFromParams{Actor: id, Target: l.UserID}); n > 0 {
			myInterest = true
		}
		if mutual {
			if p, found, err := profile.Get(r.Context(), s.DB, l.UserID); err == nil && found {
				contact, contactSelf = p.ContactHandle, p.ContactSelfConfirmed
			}
		}
	}
	stayRevs, visitRevs := []reviewView{}, []reviewView{}
	var ratings, weights []int64
	if rows, err := database.New(s.DB).ListListingReviews(r.Context(), l.ID); err == nil {
		for _, rv := range rows {
			v := reviewView{AuthorName: s.authorName(r, rv.Author), Rating: rv.Rating, Body: rv.Body}
			w := int64(1)
			if rv.Kind == "stay" {
				stayRevs = append(stayRevs, v)
				w = 3
			} else {
				visitRevs = append(visitRevs, v)
			}
			ratings = append(ratings, rv.Rating)
			weights = append(weights, w)
		}
	}
	data := map[string]any{"Title": l.Title, "Authed": true,
		"L": views[0], "Mine": l.UserID == id, "CSRF": s.csrfToken(w, r),
		"Mutual": mutual, "MyInterest": myInterest, "Contact": contact, "ContactSelf": contactSelf,
		"StayReviews": stayRevs, "VisitReviews": visitRevs,
		"TrustScore": strconv.FormatFloat(trustScore(ratings, weights), 'f', 1, 64),
		"CanStay":    s.tenantStay(r, l.ID, id)}
	if errMsg != "" {
		data["Error"] = errMsg
	}
	s.render(w, r, "listing", data)
}

func (s *Server) handleListingShow(w http.ResponseWriter, r *http.Request) {
	id := authedID(r)
	l, ok := s.listingOwned(w, r, id, false)
	if !ok {
		return
	}
	s.renderListing(w, r, l, "")
}

func (s *Server) handleListingEdit(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRF(w, r) {
		return
	}
	id := authedID(r)
	l, ok := s.listingOwned(w, r, id, true)
	if !ok {
		return
	}
	title := strings.TrimSpace(r.FormValue("title"))
	description := strings.TrimSpace(r.FormValue("description"))
	area := strings.TrimSpace(r.FormValue("area_text"))
	currency := strings.ToUpper(strings.TrimSpace(r.FormValue("currency")))
	if currency == "" {
		currency = "USD"
	}
	rent, err := parseMoney(r.FormValue("monthly_rent"))
	if err != nil {
		s.renderListing(w, r, l, "Rent must be a plain number, 0 or more.")
		return
	}
	photo := strings.TrimSpace(r.FormValue("photo_url"))
	if err := checkListing(title, description, area, currency, rent); err != nil {
		s.renderListing(w, r, l, err.Error())
		return
	}
	if err := checkPhotoURL(photo); err != nil {
		s.renderListing(w, r, l, err.Error())
		return
	}
	if err := database.New(s.DB).UpdateListing(r.Context(), database.UpdateListingParams{
		Title: title, Description: description, AreaText: area,
		MonthlyRent: rent, Currency: currency, PhotoUrl: photo, UpdatedAt: nowRFC3339(),
		ID: l.ID, UserID: id,
	}); err != nil {
		s.renderListing(w, r, l, "Something went wrong. Try again.")
		return
	}
	http.Redirect(w, r, "/listing/"+strconv.FormatInt(l.ID, 10), http.StatusSeeOther)
}

func (s *Server) handleListingDelete(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRF(w, r) {
		return
	}
	id := authedID(r)
	l, ok := s.listingOwned(w, r, id, true)
	if !ok {
		return
	}
	database.New(s.DB).DeleteListing(r.Context(), database.DeleteListingParams{ID: l.ID, UserID: id})
	http.Redirect(w, r, "/listings", http.StatusSeeOther)
}

// handleListingReport reports the listing owner through the standard
// user-report flow.
func (s *Server) handleListingReport(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRF(w, r) {
		return
	}
	id := authedID(r)
	l, ok := s.listingOwned(w, r, id, false)
	if !ok {
		return
	}
	target := l.UserID
	if target == id {
		http.NotFound(w, r)
		return
	}
	reason := strings.TrimSpace(r.FormValue("reason"))
	if reason == "" {
		reason = "No reason given"
	}
	if len(reason) > 500 {
		reason = reason[:500]
	}
	n, _ := database.New(s.DB).CountReport(r.Context(), database.CountReportParams{Reporter: id, Target: target})
	if n == 0 {
		database.New(s.DB).AddReport(r.Context(), database.AddReportParams{
			Reporter: id, Target: target, Reason: reason, CreatedAt: nowRFC3339(),
		})
	}
	http.Redirect(w, r, "/listing/"+strconv.FormatInt(l.ID, 10), http.StatusSeeOther)
}

func (s *Server) handleChatShow(w http.ResponseWriter, r *http.Request) {
	id := authedID(r)
	items, err := s.inbox(r, id)
	if err != nil {
		http.Error(w, "load error", http.StatusInternalServerError)
		return
	}
	views := []convoView{}
	for _, it := range items {
		views = append(views, convoView{
			PeerID: it.PeerID, PeerName: it.PeerName,
			LastBody: it.LastBody, LastAt: it.LastAt, LastFromMe: it.LastFromMe,
			Unread: it.Unread,
		})
	}
	s.render(w, r, "chat", map[string]any{"Title": "Chats", "Authed": true,
		"Conversations": views, "CSRF": s.csrfToken(w, r)})
}

type alertView struct {
	Kind      string
	ActorID   int64
	ActorName string
	RefID     int64
	Body      string
	Unread    bool
	At        string
}

func (s *Server) handleAlertsShow(w http.ResponseWriter, r *http.Request) {
	id := authedID(r)
	rows, err := database.New(s.DB).ListNotifications(r.Context(), database.ListNotificationsParams{UserID: id, Limit: 30})
	if err != nil {
		http.Error(w, "load error", http.StatusInternalServerError)
		return
	}
	views := []alertView{}
	for _, n := range rows {
		views = append(views, alertView{Kind: n.Kind, ActorID: n.ActorID, ActorName: s.authorName(r, n.ActorID), RefID: n.RefID, Body: n.Body, Unread: !n.ReadAt.Valid, At: n.CreatedAt})
	}
	s.render(w, r, "alerts", map[string]any{"Title": "Alerts", "Authed": true, "Alerts": views, "CSRF": s.csrfToken(w, r)})
}

func (s *Server) handleAlertsRead(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRF(w, r) {
		return
	}
	database.New(s.DB).MarkNotificationsRead(r.Context(), database.MarkNotificationsReadParams{ReadAt: sql.NullString{String: nowRFC3339(), Valid: true}, UserID: authedID(r)})
	http.Redirect(w, r, "/alerts", http.StatusSeeOther)
}

// threadPeer resolves the peer for a thread view. Reading history stays
// possible after hiding; only blocks and deactivation close it.
func (s *Server) threadPeer(w http.ResponseWriter, r *http.Request, id int64) (int64, *profile.Profile, bool) {
	peer, ok := pathID(r)
	if !ok || peer == id {
		http.NotFound(w, r)
		return 0, nil, false
	}
	if s.blockedPair(r, id, peer) || s.deactivated(r, peer) {
		http.NotFound(w, r)
		return 0, nil, false
	}
	p, found, err := profile.Get(r.Context(), s.DB, peer)
	if err != nil {
		http.Error(w, "load error", http.StatusInternalServerError)
		return 0, nil, false
	}
	if !found {
		http.NotFound(w, r)
		return 0, nil, false
	}
	return peer, p, true
}

func (s *Server) renderThread(w http.ResponseWriter, r *http.Request, peer int64, p *profile.Profile, errMsg string) {
	id := authedID(r)
	rows, err := database.New(s.DB).ListConversation(r.Context(), database.ListConversationParams{
		S1: id, R1: peer, S2: peer, R2: id, Limit: 100,
	})
	if err != nil {
		http.Error(w, "load error", http.StatusInternalServerError)
		return
	}
	_ = database.New(s.DB).MarkThreadRead(r.Context(), database.MarkThreadReadParams{
		ReadAt: sql.NullString{String: nowRFC3339(), Valid: true}, Sender: peer, Recipient: id,
	})
	views := []msgView{}
	for _, m := range rows {
		views = append(views, msgView{Mine: m.Sender == id, Body: m.Body, At: m.CreatedAt})
	}
	data := map[string]any{"Title": p.FirstName, "Authed": true,
		"PeerID": peer, "PeerName": p.FirstName, "Messages": views, "CSRF": s.csrfToken(w, r)}
	if errMsg != "" {
		data["Error"] = errMsg
	}
	s.render(w, r, "thread", data)
}

func (s *Server) handleThreadShow(w http.ResponseWriter, r *http.Request) {
	id := authedID(r)
	peer, p, ok := s.threadPeer(w, r, id)
	if !ok {
		return
	}
	s.renderThread(w, r, peer, p, "")
}

func (s *Server) handleThreadPost(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRF(w, r) {
		return
	}
	id := authedID(r)
	peer, p, ok := s.threadPeer(w, r, id)
	if !ok {
		return
	}
	if code, msg := s.messageGate(r, id, peer); code != 0 {
		if code == http.StatusNotFound {
			http.NotFound(w, r)
			return
		}
		s.renderThread(w, r, peer, p, msg)
		return
	}
	body := strings.TrimSpace(r.FormValue("body"))
	if err := checkMessage(body); err != nil {
		s.renderThread(w, r, peer, p, err.Error())
		return
	}
	if !s.MsgLimiter.Allow("user:" + strconv.FormatInt(id, 10)) {
		s.renderThread(w, r, peer, p, "Slow down a touch and try again.")
		return
	}
	if err := database.New(s.DB).InsertMessage(r.Context(), database.InsertMessageParams{
		Sender: id, Recipient: peer, Body: body, CreatedAt: nowRFC3339(),
	}); err != nil {
		s.renderThread(w, r, peer, p, "Something went wrong. Try again.")
		return
	}
	http.Redirect(w, r, "/chat/"+strconv.FormatInt(peer, 10), http.StatusSeeOther)
}

// handleReviewPost saves the caller's review of a match, then returns
// to the detail page. Failures ride back as an ?err message.
func (s *Server) handleReviewPost(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRF(w, r) {
		return
	}
	id := authedID(r)
	target, ok := pathID(r)
	back := "/match/" + r.PathValue("id")
	fail := func(msg string) {
		http.Redirect(w, r, back+"?err="+url.QueryEscape(msg), http.StatusSeeOther)
	}
	if !ok || target == id {
		http.NotFound(w, r)
		return
	}
	if s.blockedPair(r, id, target) || s.deactivated(r, target) {
		http.NotFound(w, r)
		return
	}
	rating, err := strconv.ParseInt(strings.TrimSpace(r.FormValue("rating")), 10, 64)
	if err != nil {
		fail("Rating must be 1–5.")
		return
	}
	body := strings.TrimSpace(r.FormValue("body"))
	if err := checkReview(rating, body); err != nil {
		fail(err.Error())
		return
	}
	if !s.verifiedStay(r, id, target) {
		fail("You can review after a verified stay together.")
		return
	}
	if err := database.New(s.DB).UpsertReview(r.Context(), database.UpsertReviewParams{
		Author: id, Target: target, Rating: rating, Body: body, CreatedAt: nowRFC3339(),
	}); err != nil {
		fail("Something went wrong. Try again.")
		return
	}
	s.notify(r, target, "user_review", id, id, s.authorName(r, id)+" left you a review.")
	http.Redirect(w, r, back, http.StatusSeeOther)
}

func (s *Server) handleStaysPropose(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRF(w, r) {
		return
	}
	id := authedID(r)
	peer, _ := strconv.ParseInt(strings.TrimSpace(r.FormValue("peer_id")), 10, 64)
	listingID, _ := strconv.ParseInt(strings.TrimSpace(r.FormValue("listing_id")), 10, 64)
	back := "/match/" + strconv.FormatInt(peer, 10)
	fail := func(msg string) {
		http.Redirect(w, r, back+"?err="+url.QueryEscape(msg), http.StatusSeeOther)
	}
	if peer == 0 || peer == id {
		http.NotFound(w, r)
		return
	}
	if code, msg := s.messageGate(r, id, peer); code != 0 {
		if code == http.StatusNotFound {
			http.NotFound(w, r)
			return
		}
		fail(msg)
		return
	}
	var listing sql.NullInt64
	if listingID > 0 {
		l, err := database.New(s.DB).GetListing(r.Context(), listingID)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if l.UserID != id && l.UserID != peer {
			fail("That room belongs to neither of you.")
			return
		}
		listing = sql.NullInt64{Int64: listingID, Valid: true}
	}
	a, b := canonPair(id, peer)
	now := nowRFC3339()
	if st, err := database.New(s.DB).GetStayBetween(r.Context(), database.GetStayBetweenParams{A: a, B: b}); err == nil {
		if st.Status != StayDeclined {
			fail("A stay is already in progress.")
			return
		}
		if err := database.New(s.DB).UpdateStayProposal(r.Context(), database.UpdateStayProposalParams{
			ProposedBy: id, ListingID: listing, Status: StayProposed, UpdatedAt: now, ID: st.ID,
		}); err != nil {
			fail("Something went wrong. Try again.")
			return
		}
		other := a
		if other == id {
			other = b
		}
		s.notify(r, other, "stay_proposed", id, st.ID, s.authorName(r, id)+" wants to confirm you live together.")
		http.Redirect(w, r, back, http.StatusSeeOther)
		return
	}
	res, err := database.New(s.DB).CreateStay(r.Context(), database.CreateStayParams{
		UserA: a, UserB: b, ProposedBy: id, ListingID: listing,
		Status: StayProposed, CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		fail("Something went wrong. Try again.")
		return
	}
	stayID, _ := res.LastInsertId()
	otherID := a
	if otherID == id {
		otherID = b
	}
	s.notify(r, otherID, "stay_proposed", id, stayID, s.authorName(r, id)+" wants to confirm you live together.")
	http.Redirect(w, r, back, http.StatusSeeOther)
}

func (s *Server) handleStayTransition(w http.ResponseWriter, r *http.Request, action string) {
	if !s.checkCSRF(w, r) {
		return
	}
	id := authedID(r)
	stayID, ok := pathID(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	st, err := database.New(s.DB).GetStay(r.Context(), stayID)
	if err != nil || (st.UserA != id && st.UserB != id) {
		http.NotFound(w, r)
		return
	}
	peer := st.UserA
	if peer == id {
		peer = st.UserB
	}
	back := "/match/" + strconv.FormatInt(peer, 10)
	if b := strings.TrimSpace(r.FormValue("back")); strings.HasPrefix(b, "/") && !strings.HasPrefix(b, "//") {
		back = b
	}
	fail := func(msg string) {
		http.Redirect(w, r, back+"?err="+url.QueryEscape(msg), http.StatusSeeOther)
	}
	next := ""
	switch action {
	case "accept", "decline":
		if st.Status != StayProposed || st.ProposedBy == id {
			fail("Nothing to respond to.")
			return
		}
		if action == "accept" {
			next = StayActive
		} else {
			next = StayDeclined
		}
	case "end":
		if st.Status != StayActive {
			fail("No active stay to end.")
			return
		}
		next = StayEnded
	}
	if err := database.New(s.DB).UpdateStayStatus(r.Context(), database.UpdateStayStatusParams{
		Status: next, UpdatedAt: nowRFC3339(), ID: st.ID,
	}); err != nil {
		fail("Something went wrong. Try again.")
		return
	}
	other := st.UserA
	if other == id {
		other = st.UserB
	}
	switch next {
	case StayActive:
		s.notify(r, other, "stay_accepted", id, st.ID, s.authorName(r, id)+" confirmed your stay.")
	case StayDeclined:
		s.notify(r, other, "stay_declined", id, st.ID, s.authorName(r, id)+" declined your stay proposal.")
	case StayEnded:
		s.notify(r, other, "stay_ended", id, st.ID, s.authorName(r, id)+" ended your stay.")
	}
	if next == StayActive && st.ListingID.Valid {
		_ = database.New(s.DB).UpdateListingStatus(r.Context(), database.UpdateListingStatusParams{
			Status: "rented", UpdatedAt: nowRFC3339(), ID: st.ListingID.Int64,
		})
	}
	http.Redirect(w, r, back, http.StatusSeeOther)
}

func (s *Server) handleStayAccept(w http.ResponseWriter, r *http.Request) {
	s.handleStayTransition(w, r, "accept")
}

func (s *Server) handleStayDecline(w http.ResponseWriter, r *http.Request) {
	s.handleStayTransition(w, r, "decline")
}

func (s *Server) handleStayEnd(w http.ResponseWriter, r *http.Request) {
	s.handleStayTransition(w, r, "end")
}

func (s *Server) handleListingReviewsPost(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRF(w, r) {
		return
	}
	id := authedID(r)
	l, ok := s.listingOwned(w, r, id, false)
	if !ok {
		return
	}
	back := "/listing/" + strconv.FormatInt(l.ID, 10)
	fail := func(msg string) { s.renderListing(w, r, l, msg) }
	if l.UserID == id {
		fail("You can't review your own listing.")
		return
	}
	kind := strings.ToLower(strings.TrimSpace(r.FormValue("kind")))
	if kind != "stay" && kind != "visit" {
		fail("Choose stay or visit.")
		return
	}
	if kind == "stay" && !s.tenantStay(r, l.ID, id) {
		fail("Only verified tenants can leave a stay review.")
		return
	}
	rating, err := strconv.ParseInt(strings.TrimSpace(r.FormValue("rating")), 10, 64)
	if err != nil {
		fail("Rating must be 1–5.")
		return
	}
	body := strings.TrimSpace(r.FormValue("body"))
	if err := checkReview(rating, body); err != nil {
		fail(err.Error())
		return
	}
	if err := database.New(s.DB).UpsertListingReview(r.Context(), database.UpsertListingReviewParams{
		ListingID: l.ID, Author: id, Rating: rating, Body: body, Kind: kind, CreatedAt: nowRFC3339(),
	}); err != nil {
		fail("Something went wrong. Try again.")
		return
	}
	s.notify(r, l.UserID, "listing_review", id, l.ID, s.authorName(r, id)+" left a "+kind+" review on "+l.Title+".")
	http.Redirect(w, r, back, http.StatusSeeOther)
}

func (s *Server) handleListingStatus(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRF(w, r) {
		return
	}
	id := authedID(r)
	l, ok := s.listingOwned(w, r, id, true)
	if !ok {
		return
	}
	status := strings.ToLower(strings.TrimSpace(r.FormValue("status")))
	if status != "available" && status != "rented" {
		s.renderListing(w, r, l, "Status must be available or rented.")
		return
	}
	if err := database.New(s.DB).UpdateListingStatus(r.Context(), database.UpdateListingStatusParams{
		Status: status, UpdatedAt: nowRFC3339(), ID: l.ID,
	}); err != nil {
		s.renderListing(w, r, l, "Something went wrong. Try again.")
		return
	}
	l.Status = status
	s.renderListing(w, r, l, "")
}
