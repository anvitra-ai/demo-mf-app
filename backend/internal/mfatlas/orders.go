package mfatlas

import "context"

type CreateOrderRequest struct {
	OrderType           string  `json:"order_type"` // LUMPSUM_PURCHASE (only type this app uses)
	InvestorID          string  `json:"investor_id"`
	InvestmentAccountID string  `json:"investment_account_id"`
	SchemeCode          string  `json:"scheme_code"`
	Amount              float64 `json:"amount"`
	PurchaseType        string  `json:"purchase_type"` // FRESH|ADDITIONAL
	BankAccountID       string  `json:"bank_account_id,omitempty"`
	MandateID           string  `json:"mandate_id,omitempty"`
	ClientRef           string  `json:"client_ref,omitempty"`
}

type Order struct {
	ID                  string  `json:"id"`
	InvestorID          string  `json:"investor_id"`
	InvestmentAccountID string  `json:"investment_account_id"`
	SchemeCode          string  `json:"scheme_code"`
	OrderType           string  `json:"order_type"`
	Status              string  `json:"status"`
	Amount              float64 `json:"amount"`
	Units               float64 `json:"units"`
	NAV                 float64 `json:"nav"`
	FolioNo             string  `json:"folio_no"`
	AllotmentDate       string  `json:"allotment_date"`
	PaymentStatus       string  `json:"payment_status"`
	Provider            string  `json:"provider"`
	ProviderOrderID     string  `json:"provider_order_id"`
	ProviderRemark      string  `json:"provider_remark"`
	ClientRef           string  `json:"client_ref"`
	CreatedAt           string  `json:"created_at"`
	UpdatedAt           string  `json:"updated_at"`
}

type OrderEvent struct {
	From   string `json:"from"`
	To     string `json:"to"`
	At     string `json:"at"`
	Remark string `json:"remark"`
}

func (c *Client) CreateOrder(ctx context.Context, req CreateOrderRequest, idempotencyKey string) (*Order, error) {
	var out Order
	if err := c.do(ctx, "POST", "/api/orders/v1/", req, &out, WithIdempotencyKey(idempotencyKey)); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListOrders(ctx context.Context, investorID string) ([]Order, error) {
	var out []Order
	opts := []RequestOption{}
	if investorID != "" {
		opts = append(opts, WithQuery(map[string]string{"investor_id": investorID}))
	}
	if err := c.do(ctx, "GET", "/api/orders/v1/", nil, &out, opts...); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetOrder(ctx context.Context, id string) (*Order, error) {
	var out Order
	if err := c.do(ctx, "GET", "/api/orders/v1/"+id, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetOrderEvents(ctx context.Context, id string) ([]OrderEvent, error) {
	var out []OrderEvent
	if err := c.do(ctx, "GET", "/api/orders/v1/"+id+"/events", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}
