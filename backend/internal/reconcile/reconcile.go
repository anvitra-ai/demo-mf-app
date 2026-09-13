// Package reconcile runs a low-frequency polling loop that catches anything
// a webhook missed (lost delivery, PARKED, or webhooks not configured at
// all) — guide §5.4 step 3. It re-fetches every locally non-terminal
// investor/mandate/order/payment.
package reconcile

import (
	"context"
	"log"
	"time"

	"demo-mf-app/internal/mfatlas"
	"demo-mf-app/internal/models"
	"demo-mf-app/internal/store"
)

type Poller struct {
	MF    *mfatlas.Client
	Store *store.Store
}

func (p *Poller) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.tick(ctx)
		}
	}
}

func (p *Poller) tick(ctx context.Context) {
	if investors, err := p.Store.ListNonTerminalInvestors(ctx); err == nil {
		for i := range investors {
			doc := investors[i]
			inv, err := p.MF.GetInvestor(ctx, doc.ID, false)
			if err != nil {
				log.Printf("reconcile: investor %s: %v", doc.ID, err)
				continue
			}
			updated := investorSnapshot(inv, &doc)
			_ = p.Store.UpsertInvestor(ctx, updated)
		}
	}

	if mandates, err := p.Store.ListNonTerminalMandates(ctx); err == nil {
		for i := range mandates {
			doc := mandates[i]
			m, err := p.MF.GetMandate(ctx, doc.ID)
			if err != nil {
				log.Printf("reconcile: mandate %s: %v", doc.ID, err)
				continue
			}
			updated := mandateSnapshot(m, &doc)
			_ = p.Store.UpsertMandate(ctx, updated)
		}
	}

	if orders, err := p.Store.ListNonTerminalOrders(ctx); err == nil {
		for i := range orders {
			doc := orders[i]
			o, err := p.MF.GetOrder(ctx, doc.ID)
			if err != nil {
				log.Printf("reconcile: order %s: %v", doc.ID, err)
				continue
			}
			updated := orderSnapshot(o, &doc)
			_ = p.Store.UpsertOrder(ctx, updated)
		}
	}

	if payments, err := p.Store.ListNonTerminalPayments(ctx); err == nil {
		for i := range payments {
			doc := payments[i]
			pay, err := p.MF.GetPayment(ctx, doc.ID)
			if err != nil {
				log.Printf("reconcile: payment %s: %v", doc.ID, err)
				continue
			}
			updated := paymentSnapshot(pay, &doc)
			_ = p.Store.UpsertPayment(ctx, updated)
		}
	}
}

// The snapshot helpers below mirror internal/api's *ToDoc helpers, kept
// separate to avoid an import cycle between api and reconcile.

func investorSnapshot(inv *mfatlas.Investor, prev *models.InvestorDoc) *models.InvestorDoc {
	doc := *prev
	doc.Status = inv.Status
	doc.AuthLink = inv.AuthLink
	doc.UpdatedAt = time.Now().UTC()
	if len(inv.ProviderSteps) > 0 {
		doc.ProviderRemark = inv.ProviderSteps[len(inv.ProviderSteps)-1].Remark
	}
	doc.RawLastResponse = inv
	return &doc
}

func mandateSnapshot(m *mfatlas.Mandate, prev *models.MandateDoc) *models.MandateDoc {
	doc := *prev
	doc.Status = m.Status
	doc.AuthLink = m.AuthLink
	doc.UMRN = m.UMRN
	doc.Remark = m.Remark
	doc.UpdatedAt = time.Now().UTC()
	return &doc
}

func orderSnapshot(o *mfatlas.Order, prev *models.OrderDoc) *models.OrderDoc {
	doc := *prev
	doc.Status = o.Status
	doc.Units = o.Units
	doc.NAV = o.NAV
	doc.PaymentStatus = o.PaymentStatus
	doc.ProviderRemark = o.ProviderRemark
	doc.UpdatedAt = time.Now().UTC()
	return &doc
}

func paymentSnapshot(pay *mfatlas.Payment, prev *models.PaymentDoc) *models.PaymentDoc {
	doc := *prev
	doc.Status = pay.Status
	doc.PaymentURL = pay.PaymentURL
	doc.UpdatedAt = time.Now().UTC()
	return &doc
}
