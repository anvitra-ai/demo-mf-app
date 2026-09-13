package api

import (
	"net/http"
	"time"

	"demo-mf-app/internal/mfatlas"
	"demo-mf-app/internal/models"
)

type createAccountBody struct {
	InvestorID string `json:"investor_id"`
	Code       string `json:"code"`
	Name       string `json:"name"`
}

func (a *App) handleCreateAccount(w http.ResponseWriter, r *http.Request) {
	var body createAccountBody
	if err := readJSON(r, &body); err != nil {
		writeBadRequest(w, "invalid request body: "+err.Error())
		return
	}
	if body.InvestorID == "" || body.Code == "" || body.Name == "" {
		writeBadRequest(w, "investor_id, code and name are required")
		return
	}

	acc, err := a.MF.CreateInvestmentAccount(r.Context(), mfatlas.CreateInvestmentAccountRequest{
		InvestorID: body.InvestorID,
		Code:       body.Code,
		Name:       body.Name,
	})
	if err != nil {
		writeError(w, err)
		return
	}

	doc := &models.AccountDoc{
		ID:         acc.ID,
		InvestorID: acc.InvestorID,
		Code:       acc.Code,
		Name:       acc.Name,
		Status:     acc.Status,
		CreatedAt:  time.Now().UTC(),
	}
	if err := a.Store.UpsertAccount(r.Context(), doc); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, doc)
}

func (a *App) handleListAccounts(w http.ResponseWriter, r *http.Request) {
	investorID := r.URL.Query().Get("investor_id")
	docs, err := a.Store.ListAccounts(r.Context(), investorID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, docs)
}
