package mfatlas

import "context"

type CreateMandateRequest struct {
	InvestorID          string  `json:"investor_id"`
	InvestmentAccountID string  `json:"investment_account_id"`
	BankAccountID       string  `json:"bank_account_id,omitempty"`
	AccountNo           string  `json:"account_no"`
	AccountType         string  `json:"account_type"`
	IFSC                string  `json:"ifsc"`
	Type                string  `json:"type"` // ENACH|PHYSICAL
	Amount              float64 `json:"amount"`
	StartDate           string  `json:"start_date"`
	EndDate             string  `json:"end_date"`
	ClientRef           string  `json:"client_ref,omitempty"`
}

type Mandate struct {
	ID                  string  `json:"id"`
	InvestorID          string  `json:"investor_id"`
	InvestmentAccountID string  `json:"investment_account_id"`
	Type                string  `json:"type"`
	Status              string  `json:"status"`
	Amount              float64 `json:"amount"`
	StartDate           string  `json:"start_date"`
	EndDate             string  `json:"end_date"`
	UMRN                string  `json:"umrn"`
	AuthLink            string  `json:"auth_link"`
	ProviderMandateID   string  `json:"provider_mandate_id"`
	Remark              string  `json:"remark"`
	ClientRef           string  `json:"client_ref"`
	CreatedAt           string  `json:"created_at"`
	UpdatedAt           string  `json:"updated_at"`
}

func (c *Client) CreateMandate(ctx context.Context, req CreateMandateRequest) (*Mandate, error) {
	var out Mandate
	if err := c.do(ctx, "POST", "/api/mandates/v1/", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListMandates(ctx context.Context, investorID string) ([]Mandate, error) {
	var out []Mandate
	opts := []RequestOption{}
	if investorID != "" {
		opts = append(opts, WithQuery(map[string]string{"investor_id": investorID}))
	}
	if err := c.do(ctx, "GET", "/api/mandates/v1/", nil, &out, opts...); err != nil {
		return nil, err
	}
	return out, nil
}

// GetMandate refreshes status live from the exchange before returning
// (guide §3.2) — always call this fresh before using a mandate for payment.
func (c *Client) GetMandate(ctx context.Context, id string) (*Mandate, error) {
	var out Mandate
	if err := c.do(ctx, "GET", "/api/mandates/v1/"+id, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
