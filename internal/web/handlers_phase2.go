package web

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gentcod/flatmate/internal/ai"
	"github.com/gentcod/flatmate/internal/database"
	"github.com/gentcod/flatmate/internal/match"
	"github.com/gentcod/flatmate/internal/profile"
)

func (s *Server) renderExpenses(w http.ResponseWriter, r *http.Request, peer int64, peerName, errMsg string) {
	id := authedID(r)
	exp, bals, err := s.ledger(r, id, peer)
	if err != nil {
		http.Error(w, "load error", http.StatusInternalServerError)
		return
	}
	data := map[string]any{"Title": "Shared expenses", "Authed": true,
		"PeerID": peer, "PeerName": peerName, "Expenses": exp, "Balances": bals,
		"CSRF": s.csrfToken(w, r)}
	if errMsg != "" {
		data["Error"] = errMsg
	}
	s.render(w, "expenses", data)
}

func (s *Server) handleExpensesShow(w http.ResponseWriter, r *http.Request) {
	id := authedID(r)
	peer, p, ok := s.threadPeer(w, r, id)
	if !ok {
		return
	}
	s.renderExpenses(w, r, peer, p.FirstName, "")
}

func (s *Server) handleExpensesPost(w http.ResponseWriter, r *http.Request) {
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
		s.renderExpenses(w, r, peer, p.FirstName, msg)
		return
	}
	amount, err := parseMoney(r.FormValue("amount"))
	if err != nil {
		s.renderExpenses(w, r, peer, p.FirstName, "Amount must be a plain number.")
		return
	}
	currency := strings.ToUpper(strings.TrimSpace(r.FormValue("currency")))
	if currency == "" {
		currency = "USD"
	}
	note := strings.TrimSpace(r.FormValue("note"))
	split := strings.TrimSpace(r.FormValue("split"))
	if split == "" {
		split = "equal"
	}
	payer, other := id, peer
	if r.FormValue("payer") == "them" {
		payer, other = peer, id
	}
	var ps, qs int64
	if split == "custom" {
		if ps, err = parseMoney(r.FormValue("payer_share")); err != nil {
			s.renderExpenses(w, r, peer, p.FirstName, "Custom shares must be plain numbers.")
			return
		}
		if qs, err = parseMoney(r.FormValue("peer_share")); err != nil {
			s.renderExpenses(w, r, peer, p.FirstName, "Custom shares must be plain numbers.")
			return
		}
	}
	ps, qs = splitShares(split, amount, ps, qs)
	if err := checkExpense(amount, currency, note, split, ps, qs); err != nil {
		s.renderExpenses(w, r, peer, p.FirstName, err.Error())
		return
	}
	if !s.MsgLimiter.Allow("exp:user:" + strconv.FormatInt(id, 10)) {
		s.renderExpenses(w, r, peer, p.FirstName, "Slow down a touch and try again.")
		return
	}
	if err := database.New(s.DB).InsertExpense(r.Context(), database.InsertExpenseParams{
		Payer: payer, Peer: other, Amount: amount, Currency: currency,
		Note: note, Split: split, PayerShare: ps, PeerShare: qs, CreatedAt: nowRFC3339(),
	}); err != nil {
		s.renderExpenses(w, r, peer, p.FirstName, "Something went wrong. Try again.")
		return
	}
	http.Redirect(w, r, "/expenses/"+strconv.FormatInt(peer, 10), http.StatusSeeOther)
}

func (s *Server) handleExpenseDelete(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRF(w, r) {
		return
	}
	id := authedID(r)
	target, ok := pathID(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	e, err := database.New(s.DB).GetExpense(r.Context(), target)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if e.Payer != id && e.Peer != id {
		http.NotFound(w, r)
		return
	}
	other := e.Payer
	if other == id {
		other = e.Peer
	}
	if s.blockedPair(r, id, other) || s.deactivated(r, other) {
		http.NotFound(w, r)
		return
	}
	database.New(s.DB).DeleteExpense(r.Context(), e.ID)
	http.Redirect(w, r, "/expenses/"+strconv.FormatInt(other, 10), http.StatusSeeOther)
}

func (s *Server) handleSearchesPost(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRF(w, r) {
		return
	}
	id := authedID(r)
	name := strings.TrimSpace(r.FormValue("name"))
	area := strings.TrimSpace(r.FormValue("area"))
	maxRent, err := parseMoney(r.FormValue("max_rent"))
	if r.FormValue("max_rent") == "" {
		maxRent, err = 0, nil
	}
	if err != nil {
		s.renderListings(w, r, "Max rent must be a plain number.")
		return
	}
	if err := checkSearch(name, area, maxRent); err != nil {
		s.renderListings(w, r, err.Error())
		return
	}
	database.New(s.DB).CreateSearch(r.Context(), database.CreateSearchParams{
		UserID: id, Name: name, Area: area, MaxRent: maxRent, CreatedAt: nowRFC3339(),
	})
	http.Redirect(w, r, "/listings", http.StatusSeeOther)
}

func (s *Server) handleSearchDelete(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRF(w, r) {
		return
	}
	id := authedID(r)
	target, ok := pathID(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	database.New(s.DB).DeleteSearch(r.Context(), database.DeleteSearchParams{ID: target, UserID: id})
	http.Redirect(w, r, "/listings", http.StatusSeeOther)
}

func (s *Server) handleAgreementShow(w http.ResponseWriter, r *http.Request) {
	id := authedID(r)
	target, ok := pathID(r)
	if !ok || target == id {
		http.NotFound(w, r)
		return
	}
	tp, found, err := profile.Get(r.Context(), s.DB, target)
	if err != nil {
		http.Error(w, "load error", http.StatusInternalServerError)
		return
	}
	if !found || !tp.Searching || s.blockedPair(r, id, target) || s.deactivated(r, target) {
		http.NotFound(w, r)
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
	s.render(w, "agreement", map[string]any{"Title": "Roommate agreement", "Authed": true,
		"PeerID": target, "PeerName": tp.FirstName, "Sections": sections, "Enhanced": enhanced,
		"CSRF": s.csrfToken(w, r)})
}
