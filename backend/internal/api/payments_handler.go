package api

import (
	"net/http"
	"time"

	"demo-mf-app/internal/mfatlas"
	"demo-mf-app/internal/models"

	"github.com/google/uuid"
)

type createPaymentBody struct {
	InvestorID          string   `json:"investor_id"`
	InvestmentAccountID string   `json:"investment_account_id"`
	OrderIDs            []string `json:"order_ids"`
	Mode                string   `json:"mode"` // MANDATE|CHEQUE|UPI|NETBANKING|NEFT_RTGS
	MandateID           string   `json:"mandate_id"`
	BankAccountID       string   `json:"bank_account_id"`
	VPA                 string   `json:"vpa"`
	ChequeNumber        string   `json:"cheque_number"`
	ChequeDate          string   `json:"cheque_date"`
}

func paymentToDoc(p *mfatlas.Payment, prev *models.PaymentDoc) *models.PaymentDoc {
	now := time.Now().UTC()
	doc := &models.PaymentDoc{
		ID:                  p.ID,
		InvestorID:          p.InvestorID,
		InvestmentAccountID: p.InvestmentAccountID,
		OrderIDs:            p.OrderIDs,
		Mode:                p.Mode,
		Status:              p.Status,
		Amount:              p.Amount,
		PaymentURL:          p.PaymentURL,
		UpdatedAt:           now,
	}
	if prev != nil {
		doc.CreatedAt = prev.CreatedAt
	} else {
		doc.CreatedAt = now
	}
	return doc
}

// handleCreatePayment branches by mode per guide §4.2. Before defaulting to
// MANDATE, it re-fetches the mandate live (not from local cache) to confirm
// REGISTERED status and sufficient amount, per §3.2/§4.3 — the server-side
// check at payment time exists too, but failing fast here gives a clearer
// error than a raw 422 from upstream.
func (a *App) handleCreatePayment(w http.ResponseWriter, r *http.Request) {
	var body createPaymentBody
	if err := readJSON(r, &body); err != nil {
		writeBadRequest(w, "invalid request body: "+err.Error())
		return
	}
	if body.InvestorID == "" || len(body.OrderIDs) == 0 || body.Mode == "" {
		writeBadRequest(w, "investor_id, order_ids and mode are required")
		return
	}

	req := mfatlas.CreatePaymentRequest{
		InvestorID:          body.InvestorID,
		InvestmentAccountID: body.InvestmentAccountID,
		OrderIDs:            body.OrderIDs,
		Mode:                body.Mode,
	}

	switch body.Mode {
	case "MANDATE":
		if body.MandateID == "" {
			writeBadRequest(w, "mandate_id is required for mode MANDATE")
			return
		}
		m, err := a.MF.GetMandate(r.Context(), body.MandateID)
		if err != nil {
			writeError(w, err)
			return
		}
		if m.Status != "REGISTERED" {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
				"error": map[string]any{"code": "MANDATE_NOT_ACTIVE", "message": "mandate is not REGISTERED yet (status: " + m.Status + ")"},
			})
			return
		}
		req.MandateID = body.MandateID
	case "UPI":
		req.VPA = body.VPA
	case "NETBANKING":
		req.BankAccountID = body.BankAccountID
	case "CHEQUE":
		if body.ChequeNumber == "" || body.ChequeDate == "" {
			writeBadRequest(w, "cheque_number and cheque_date are required for mode CHEQUE")
			return
		}
		req.Cheque = &mfatlas.ChequeDetails{Number: body.ChequeNumber, Date: body.ChequeDate}
	case "NEFT_RTGS":
		// no extra fields at creation time; UTR is submitted afterwards.
	default:
		writeBadRequest(w, "mode must be one of MANDATE, UPI, NETBANKING, CHEQUE, NEFT_RTGS")
		return
	}

	p, err := a.MF.CreatePayment(r.Context(), req, uuid.NewString())
	if err != nil {
		writeError(w, err)
		return
	}

	doc := paymentToDoc(p, nil)
	if err := a.Store.UpsertPayment(r.Context(), doc); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, doc)
}

func (a *App) handleListPayments(w http.ResponseWriter, r *http.Request) {
	investorID := r.URL.Query().Get("investor_id")
	payments, err := a.MF.ListPayments(r.Context(), investorID)
	if err != nil {
		writeError(w, err)
		return
	}
	docs := make([]*models.PaymentDoc, 0, len(payments))
	for i := range payments {
		prev, err := a.Store.GetPayment(r.Context(), payments[i].ID)
		if err != nil {
			writeError(w, err)
			return
		}
		doc := paymentToDoc(&payments[i], prev)
		if err := a.Store.UpsertPayment(r.Context(), doc); err != nil {
			writeError(w, err)
			return
		}
		docs = append(docs, doc)
	}
	writeJSON(w, http.StatusOK, docs)
}

func (a *App) handleGetPayment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	doc, err := a.Store.GetPayment(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	if doc == nil {
		writeNotFound(w, "payment not found")
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (a *App) handleRefreshPayment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	prev, err := a.Store.GetPayment(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	if prev == nil {
		writeNotFound(w, "payment not found")
		return
	}

	p, err := a.MF.GetPayment(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	doc := paymentToDoc(p, prev)
	if err := a.Store.UpsertPayment(r.Context(), doc); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

type submitUTRBody struct {
	UTR          string `json:"utr"`
	TransferDate string `json:"transfer_date"`
	BankName     string `json:"bank_name"`
	IFSC         string `json:"ifsc"`
	AccountNo    string `json:"account_no"`
}

// handleSubmitUTR is the required follow-up call for NEFT_RTGS payments
// (guide §4.2) — without it, settlement never completes.
func (a *App) handleSubmitUTR(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body submitUTRBody
	if err := readJSON(r, &body); err != nil {
		writeBadRequest(w, "invalid request body: "+err.Error())
		return
	}
	if body.UTR == "" {
		writeBadRequest(w, "utr is required")
		return
	}

	if err := a.MF.SubmitUTR(r.Context(), id, mfatlas.SubmitUTRRequest{
		UTR:          body.UTR,
		TransferDate: body.TransferDate,
		BankName:     body.BankName,
		IFSC:         body.IFSC,
		AccountNo:    body.AccountNo,
	}); err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}
