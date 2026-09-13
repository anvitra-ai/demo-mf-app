package api

import (
	"net/http"
	"time"

	"demo-mf-app/internal/mfatlas"
	"demo-mf-app/internal/models"

	"github.com/google/uuid"
)

type createInvestorBody struct {
	Name         string                     `json:"name"`
	PAN          string                     `json:"pan"`
	DOB          string                     `json:"dob"`
	Mobile       string                     `json:"mobile"`
	Email        string                     `json:"email"`
	Gender       string                     `json:"gender"`
	Address      mfatlas.Address            `json:"address"`
	BankAccounts []mfatlas.BankAccountInput `json:"bank_accounts"`
	HoldingType  string                     `json:"holding_type"`
	TaxStatus    string                     `json:"tax_status"`
}

func (b createInvestorBody) toUpstream(clientRef string) mfatlas.CreateInvestorRequest {
	holdingType := b.HoldingType
	if holdingType == "" {
		holdingType = "SINGLE"
	}
	taxStatus := b.TaxStatus
	if taxStatus == "" {
		taxStatus = "INDIVIDUAL"
	}
	return mfatlas.CreateInvestorRequest{
		PrimaryHolder: mfatlas.PrimaryHolder{
			Name:   b.Name,
			PAN:    b.PAN,
			DOB:    b.DOB,
			Mobile: b.Mobile,
			Email:  b.Email,
			Gender: b.Gender,
		},
		Address:      b.Address,
		Contact:      mfatlas.Contact{Email: b.Email, Mobile: b.Mobile},
		BankAccounts: b.BankAccounts,
		HoldingType:  holdingType,
		TaxStatus:    taxStatus,
		ClientRef:    clientRef,
		Submit:       true,
	}
}

func investorToDoc(inv *mfatlas.Investor, prev *models.InvestorDoc) *models.InvestorDoc {
	now := time.Now().UTC()
	doc := &models.InvestorDoc{
		ID:              inv.ID,
		Status:          inv.Status,
		ClientRef:       inv.ClientRef,
		AuthLink:        inv.AuthLink,
		RawLastResponse: inv,
		UpdatedAt:       now,
	}
	if len(inv.ProviderSteps) > 0 {
		doc.ProviderRemark = inv.ProviderSteps[len(inv.ProviderSteps)-1].Remark
	}
	if prev != nil {
		doc.CreatedAt = prev.CreatedAt
		doc.Name = prev.Name
		doc.PAN = prev.PAN
		doc.Email = prev.Email
		doc.Mobile = prev.Mobile
	} else {
		doc.CreatedAt = now
	}
	if doc.Name == "" && inv.PrimaryHolder != nil {
		doc.Name = inv.PrimaryHolder.Name
		doc.PAN = inv.PrimaryHolder.PAN
		doc.Email = inv.PrimaryHolder.Email
		doc.Mobile = inv.PrimaryHolder.Mobile
	}
	return doc
}

func (a *App) handleCreateInvestor(w http.ResponseWriter, r *http.Request) {
	var body createInvestorBody
	if err := readJSON(r, &body); err != nil {
		writeBadRequest(w, "invalid request body: "+err.Error())
		return
	}
	if body.Name == "" || body.PAN == "" || body.Address.Line1 == "" || len(body.BankAccounts) == 0 {
		writeBadRequest(w, "name, pan, address.line1 and at least one bank account are required")
		return
	}

	clientRef := "inv-" + uuid.NewString()
	inv, err := a.MF.CreateInvestor(r.Context(), body.toUpstream(clientRef))
	if err != nil {
		writeError(w, err)
		return
	}

	doc := investorToDoc(inv, nil)
	doc.Name = body.Name
	doc.PAN = body.PAN
	doc.Email = body.Email
	doc.Mobile = body.Mobile

	if err := a.Store.UpsertInvestor(r.Context(), doc); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, doc)
}

func (a *App) handleListInvestors(w http.ResponseWriter, r *http.Request) {
	investors, err := a.MF.ListInvestors(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	docs := make([]*models.InvestorDoc, 0, len(investors))
	for i := range investors {
		prev, err := a.Store.GetInvestor(r.Context(), investors[i].ID)
		if err != nil {
			writeError(w, err)
			return
		}
		doc := investorToDoc(&investors[i], prev)
		if err := a.Store.UpsertInvestor(r.Context(), doc); err != nil {
			writeError(w, err)
			return
		}
		docs = append(docs, doc)
	}
	writeJSON(w, http.StatusOK, docs)
}

func (a *App) handleGetInvestor(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	doc, err := a.Store.GetInvestor(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	if doc == nil {
		writeNotFound(w, "investor not found")
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

// handleRefreshInvestor fetches live status (+auth_link, which is only ever
// live-fetched, never cached — guide §2.3) and updates the local mirror.
func (a *App) handleRefreshInvestor(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	prev, err := a.Store.GetInvestor(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	if prev == nil {
		writeNotFound(w, "investor not found")
		return
	}

	inv, err := a.MF.GetInvestor(r.Context(), id, true)
	if err != nil {
		writeError(w, err)
		return
	}

	doc := investorToDoc(inv, prev)
	if err := a.Store.UpsertInvestor(r.Context(), doc); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (a *App) handleRetryInvestor(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	prev, err := a.Store.GetInvestor(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	if prev == nil {
		writeNotFound(w, "investor not found")
		return
	}

	inv, err := a.MF.RetryInvestor(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	doc := investorToDoc(inv, prev)
	if err := a.Store.UpsertInvestor(r.Context(), doc); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

// handleResubmitInvestor is for the REJECTED -> fix -> resubmit loop (guide
// §2.4): PATCH with corrected fields and submit:true.
func (a *App) handleResubmitInvestor(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	prev, err := a.Store.GetInvestor(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	if prev == nil {
		writeNotFound(w, "investor not found")
		return
	}

	var body createInvestorBody
	if err := readJSON(r, &body); err != nil {
		writeBadRequest(w, "invalid request body: "+err.Error())
		return
	}

	inv, err := a.MF.PatchInvestor(r.Context(), id, body.toUpstream(prev.ClientRef))
	if err != nil {
		writeError(w, err)
		return
	}

	doc := investorToDoc(inv, prev)
	doc.Name = body.Name
	doc.PAN = body.PAN
	doc.Email = body.Email
	doc.Mobile = body.Mobile
	if err := a.Store.UpsertInvestor(r.Context(), doc); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

// handleSyncInvestors triggers exchange-wide reconciliation (guide §2.6),
// then refreshes every currently non-terminal investor's local mirror.
func (a *App) handleSyncInvestors(w http.ResponseWriter, r *http.Request) {
	result, err := a.MF.SyncInvestors(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}

	pending, err := a.Store.ListNonTerminalInvestors(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	for _, doc := range pending {
		inv, err := a.MF.GetInvestor(r.Context(), doc.ID, false)
		if err != nil {
			continue
		}
		updated := investorToDoc(inv, &doc)
		_ = a.Store.UpsertInvestor(r.Context(), updated)
	}

	writeJSON(w, http.StatusOK, result)
}
