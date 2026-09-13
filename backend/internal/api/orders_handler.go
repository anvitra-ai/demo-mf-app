package api

import (
	"net/http"
	"time"

	"demo-mf-app/internal/mfatlas"
	"demo-mf-app/internal/models"

	"github.com/google/uuid"
)

type createOrderBody struct {
	InvestorID          string  `json:"investor_id"`
	InvestmentAccountID string  `json:"investment_account_id"`
	SchemeCode          string  `json:"scheme_code"`
	Amount              float64 `json:"amount"`
	PurchaseType        string  `json:"purchase_type"`
	BankAccountID       string  `json:"bank_account_id"`
	MandateID           string  `json:"mandate_id"`
}

func orderToDoc(o *mfatlas.Order, prev *models.OrderDoc) *models.OrderDoc {
	now := time.Now().UTC()
	doc := &models.OrderDoc{
		ID:                  o.ID,
		InvestorID:          o.InvestorID,
		InvestmentAccountID: o.InvestmentAccountID,
		SchemeCode:          o.SchemeCode,
		OrderType:           o.OrderType,
		Status:              o.Status,
		Amount:              o.Amount,
		Units:               o.Units,
		NAV:                 o.NAV,
		PaymentStatus:       o.PaymentStatus,
		ProviderRemark:      o.ProviderRemark,
		ClientRef:           o.ClientRef,
		UpdatedAt:           now,
	}
	if prev != nil {
		doc.CreatedAt = prev.CreatedAt
		doc.IdempotencyKey = prev.IdempotencyKey
	} else {
		doc.CreatedAt = now
	}
	return doc
}

// This app only places LUMPSUM_PURCHASE orders — the guide's core flow
// (§4). SIP/STP/SWP/switch/redemption are documented as out of scope.
func (a *App) handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	var body createOrderBody
	if err := readJSON(r, &body); err != nil {
		writeBadRequest(w, "invalid request body: "+err.Error())
		return
	}
	if body.InvestorID == "" || body.InvestmentAccountID == "" || body.SchemeCode == "" || body.Amount <= 0 {
		writeBadRequest(w, "investor_id, investment_account_id, scheme_code and a positive amount are required")
		return
	}
	purchaseType := body.PurchaseType
	if purchaseType == "" {
		purchaseType = "FRESH"
	}

	clientRef := "ord-" + uuid.NewString()
	idempotencyKey := uuid.NewString()

	o, err := a.MF.CreateOrder(r.Context(), mfatlas.CreateOrderRequest{
		OrderType:           "LUMPSUM_PURCHASE",
		InvestorID:          body.InvestorID,
		InvestmentAccountID: body.InvestmentAccountID,
		SchemeCode:          body.SchemeCode,
		Amount:              body.Amount,
		PurchaseType:        purchaseType,
		BankAccountID:       body.BankAccountID,
		MandateID:           body.MandateID,
		ClientRef:           clientRef,
	}, idempotencyKey)
	if err != nil {
		writeError(w, err)
		return
	}

	doc := orderToDoc(o, nil)
	doc.IdempotencyKey = idempotencyKey
	if err := a.Store.UpsertOrder(r.Context(), doc); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, doc)
}

func (a *App) handleListOrders(w http.ResponseWriter, r *http.Request) {
	investorID := r.URL.Query().Get("investor_id")
	docs, err := a.Store.ListOrders(r.Context(), investorID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, docs)
}

func (a *App) handleGetOrder(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	doc, err := a.Store.GetOrder(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	if doc == nil {
		writeNotFound(w, "order not found")
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (a *App) handleRefreshOrder(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	prev, err := a.Store.GetOrder(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	if prev == nil {
		writeNotFound(w, "order not found")
		return
	}

	o, err := a.MF.GetOrder(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	doc := orderToDoc(o, prev)
	if err := a.Store.UpsertOrder(r.Context(), doc); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (a *App) handleGetOrderEvents(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	events, err := a.MF.GetOrderEvents(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, events)
}
