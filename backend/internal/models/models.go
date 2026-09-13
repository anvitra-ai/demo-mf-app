// Package models defines the local MongoDB mirror documents. Each one tracks
// an mf-atlas resource plus our own bookkeeping (last raw snapshot, local
// timestamps) so the frontend, webhook handler, and reconciliation poller all
// read/write the same shape.
package models

import "time"

type InvestorDoc struct {
	ID              string    `bson:"_id" json:"id"` // mf-atlas investor id, e.g. I_SI_IND_000042
	Status          string    `bson:"status" json:"status"`
	ClientRef       string    `bson:"client_ref" json:"client_ref"`
	Name            string    `bson:"name" json:"name"`
	PAN             string    `bson:"pan" json:"pan"`
	Email           string    `bson:"email" json:"email"`
	Mobile          string    `bson:"mobile" json:"mobile"`
	AuthLink        string    `bson:"auth_link,omitempty" json:"auth_link,omitempty"`
	ProviderRemark  string    `bson:"provider_remark,omitempty" json:"provider_remark,omitempty"`
	RawLastResponse any       `bson:"raw_last_response" json:"-"`
	CreatedAt       time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt       time.Time `bson:"updated_at" json:"updated_at"`
}

type AccountDoc struct {
	ID         string    `bson:"_id" json:"id"` // A_RT_000001
	InvestorID string    `bson:"investor_id" json:"investor_id"`
	Code       string    `bson:"code" json:"code"`
	Name       string    `bson:"name" json:"name"`
	Status     string    `bson:"status" json:"status"`
	CreatedAt  time.Time `bson:"created_at" json:"created_at"`
}

type MandateDoc struct {
	ID                  string    `bson:"_id" json:"id"` // mnd_...
	InvestorID          string    `bson:"investor_id" json:"investor_id"`
	InvestmentAccountID string    `bson:"investment_account_id" json:"investment_account_id"`
	Type                string    `bson:"type" json:"type"`
	Status              string    `bson:"status" json:"status"`
	Amount              float64   `bson:"amount" json:"amount"`
	StartDate           string    `bson:"start_date" json:"start_date"`
	EndDate             string    `bson:"end_date" json:"end_date"`
	AuthLink            string    `bson:"auth_link,omitempty" json:"auth_link,omitempty"`
	UMRN                string    `bson:"umrn,omitempty" json:"umrn,omitempty"`
	Remark              string    `bson:"remark,omitempty" json:"remark,omitempty"`
	CreatedAt           time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt           time.Time `bson:"updated_at" json:"updated_at"`
}

type OrderDoc struct {
	ID                  string    `bson:"_id" json:"id"` // ord_...
	InvestorID          string    `bson:"investor_id" json:"investor_id"`
	InvestmentAccountID string    `bson:"investment_account_id" json:"investment_account_id"`
	SchemeCode          string    `bson:"scheme_code" json:"scheme_code"`
	OrderType           string    `bson:"order_type" json:"order_type"`
	Status              string    `bson:"status" json:"status"`
	Amount              float64   `bson:"amount" json:"amount"`
	Units               float64   `bson:"units" json:"units"`
	NAV                 float64   `bson:"nav" json:"nav"`
	PaymentStatus       string    `bson:"payment_status" json:"payment_status"`
	ProviderRemark      string    `bson:"provider_remark,omitempty" json:"provider_remark,omitempty"`
	ClientRef           string    `bson:"client_ref" json:"client_ref"`
	IdempotencyKey      string    `bson:"idempotency_key" json:"-"`
	CreatedAt           time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt           time.Time `bson:"updated_at" json:"updated_at"`
}

type PaymentDoc struct {
	ID                  string    `bson:"_id" json:"id"` // pay_...
	InvestorID          string    `bson:"investor_id" json:"investor_id"`
	InvestmentAccountID string    `bson:"investment_account_id" json:"investment_account_id"`
	OrderIDs            []string  `bson:"order_ids" json:"order_ids"`
	Mode                string    `bson:"mode" json:"mode"`
	Status              string    `bson:"status" json:"status"`
	Amount              float64   `bson:"amount" json:"amount"`
	PaymentURL          string    `bson:"payment_url,omitempty" json:"payment_url,omitempty"`
	CreatedAt           time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt           time.Time `bson:"updated_at" json:"updated_at"`
}

// WebhookDeliveryDoc records the X-MF-Delivery id of every webhook we've
// processed, so at-least-once delivery (guide §5.3) can be de-duplicated.
type WebhookDeliveryDoc struct {
	ID         string    `bson:"_id" json:"id"` // X-MF-Delivery value
	Event      string    `bson:"event" json:"event"`
	ReceivedAt time.Time `bson:"received_at" json:"received_at"`
}

// NonTerminal status sets, used by the reconciliation poller.
var (
	InvestorNonTerminal = map[string]bool{"PENDING": true, "PROCESSING": true}
	MandateNonTerminal  = map[string]bool{"PENDING": true}
	OrderNonTerminal    = map[string]bool{"PENDING": true, "SUBMITTED": true, "ACCEPTED": true}
	PaymentNonTerminal  = map[string]bool{"INITIATED": true}
)
