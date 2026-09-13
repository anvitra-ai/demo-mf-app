package api

import (
	"net/http"
	"time"

	"demo-mf-app/internal/mfatlas"
	"demo-mf-app/internal/models"

	"github.com/google/uuid"
)

type createMandateBody struct {
	InvestorID          string  `json:"investor_id"`
	InvestmentAccountID string  `json:"investment_account_id"`
	BankAccountID       string  `json:"bank_account_id"`
	AccountNo           string  `json:"account_no"`
	AccountType         string  `json:"account_type"`
	IFSC                string  `json:"ifsc"`
	Amount              float64 `json:"amount"`
	StartDate           string  `json:"start_date"`
	EndDate             string  `json:"end_date"`
}

func mandateToDoc(m *mfatlas.Mandate, prev *models.MandateDoc) *models.MandateDoc {
	now := time.Now().UTC()
	doc := &models.MandateDoc{
		ID:                  m.ID,
		InvestorID:          m.InvestorID,
		InvestmentAccountID: m.InvestmentAccountID,
		Type:                m.Type,
		Status:              m.Status,
		Amount:              m.Amount,
		StartDate:           m.StartDate,
		EndDate:             m.EndDate,
		AuthLink:            m.AuthLink,
		UMRN:                m.UMRN,
		Remark:              m.Remark,
		UpdatedAt:           now,
	}
	if prev != nil {
		doc.CreatedAt = prev.CreatedAt
	} else {
		doc.CreatedAt = now
	}
	return doc
}

func (a *App) handleCreateMandate(w http.ResponseWriter, r *http.Request) {
	var body createMandateBody
	if err := readJSON(r, &body); err != nil {
		writeBadRequest(w, "invalid request body: "+err.Error())
		return
	}
	if body.InvestorID == "" || body.InvestmentAccountID == "" || body.AccountNo == "" || body.IFSC == "" || body.Amount <= 0 {
		writeBadRequest(w, "investor_id, investment_account_id, account_no, ifsc and a positive amount are required")
		return
	}

	m, err := a.MF.CreateMandate(r.Context(), mfatlas.CreateMandateRequest{
		InvestorID:          body.InvestorID,
		InvestmentAccountID: body.InvestmentAccountID,
		BankAccountID:       body.BankAccountID,
		AccountNo:           body.AccountNo,
		AccountType:         body.AccountType,
		IFSC:                body.IFSC,
		Type:                "ENACH",
		Amount:              body.Amount,
		StartDate:           body.StartDate,
		EndDate:             body.EndDate,
		ClientRef:           "mnd-" + uuid.NewString(),
	})
	if err != nil {
		writeError(w, err)
		return
	}

	doc := mandateToDoc(m, nil)
	if err := a.Store.UpsertMandate(r.Context(), doc); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, doc)
}

func (a *App) handleListMandates(w http.ResponseWriter, r *http.Request) {
	investorID := r.URL.Query().Get("investor_id")
	docs, err := a.Store.ListMandates(r.Context(), investorID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, docs)
}

func (a *App) handleGetMandate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	doc, err := a.Store.GetMandate(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	if doc == nil {
		writeNotFound(w, "mandate not found")
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

// handleRefreshMandate always hits mf-atlas live: GET /mandates/:id refreshes
// status from the exchange server-side (guide §3.2) — never trust a cached
// mandate status before using it for payment.
func (a *App) handleRefreshMandate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	prev, err := a.Store.GetMandate(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	if prev == nil {
		writeNotFound(w, "mandate not found")
		return
	}

	m, err := a.MF.GetMandate(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	doc := mandateToDoc(m, prev)
	if err := a.Store.UpsertMandate(r.Context(), doc); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, doc)
}
