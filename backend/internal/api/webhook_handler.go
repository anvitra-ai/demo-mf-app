package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"demo-mf-app/internal/mfatlas"
)

type webhookEnvelope struct {
	Success bool `json:"success"`
	Data    struct {
		InvestorID string `json:"investor_id"`
		MandateID  string `json:"mandate_id"`
		OrderID    string `json:"order_id"`
		PaymentID  string `json:"payment_id"`
		Status     string `json:"status"`
	} `json:"data"`
}

// handleWebhook implements guide §5.3/§5.4: verify signature, de-duplicate on
// X-MF-Delivery (delivery is at-least-once and can arrive out of order), then
// treat the payload as a thin trigger to re-fetch the authoritative resource
// rather than trusting the payload's own status ordering.
func (a *App) handleWebhook(w http.ResponseWriter, r *http.Request) {
	rawBody, err := io.ReadAll(r.Body)
	if err != nil {
		writeBadRequest(w, "could not read body")
		return
	}

	if a.Cfg.WebhookSecret != "" {
		sigHeader := r.Header.Get("X-MF-Signature")
		if !mfatlas.VerifySignature(sigHeader, string(rawBody), a.Cfg.WebhookSecret) {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": map[string]any{"code": "INVALID_SIGNATURE"}})
			return
		}
	} else {
		log.Println("WARNING: MF_ATLAS_WEBHOOK_SECRET not set; skipping signature verification")
	}

	deliveryID := r.Header.Get("X-MF-Delivery")
	event := r.Header.Get("X-MF-Event")

	if deliveryID != "" {
		alreadyProcessed, err := a.Store.MarkDeliveryProcessed(r.Context(), deliveryID, event)
		if err != nil {
			writeError(w, err)
			return
		}
		if alreadyProcessed {
			writeJSON(w, http.StatusOK, map[string]any{"deduped": true})
			return
		}
	}

	var env webhookEnvelope
	if err := json.Unmarshal(rawBody, &env); err != nil {
		writeBadRequest(w, "invalid payload")
		return
	}

	ctx := r.Context()
	switch {
	case strings.HasPrefix(event, "investor."):
		if env.Data.InvestorID == "" {
			break
		}
		prev, _ := a.Store.GetInvestor(ctx, env.Data.InvestorID)
		if inv, err := a.MF.GetInvestor(ctx, env.Data.InvestorID, false); err == nil {
			_ = a.Store.UpsertInvestor(ctx, investorToDoc(inv, prev))
		}
	case strings.HasPrefix(event, "mandate."):
		if env.Data.MandateID == "" {
			break
		}
		prev, _ := a.Store.GetMandate(ctx, env.Data.MandateID)
		if m, err := a.MF.GetMandate(ctx, env.Data.MandateID); err == nil {
			_ = a.Store.UpsertMandate(ctx, mandateToDoc(m, prev))
		}
	case strings.HasPrefix(event, "order."):
		if env.Data.OrderID == "" {
			break
		}
		prev, _ := a.Store.GetOrder(ctx, env.Data.OrderID)
		if o, err := a.MF.GetOrder(ctx, env.Data.OrderID); err == nil {
			_ = a.Store.UpsertOrder(ctx, orderToDoc(o, prev))
		}
	case strings.HasPrefix(event, "payment."):
		if env.Data.PaymentID == "" {
			break
		}
		prev, _ := a.Store.GetPayment(ctx, env.Data.PaymentID)
		if p, err := a.MF.GetPayment(ctx, env.Data.PaymentID); err == nil {
			_ = a.Store.UpsertPayment(ctx, paymentToDoc(p, prev))
		}
	default:
		log.Printf("unhandled webhook event: %s", event)
	}

	writeJSON(w, http.StatusOK, map[string]any{"received": true})
}
