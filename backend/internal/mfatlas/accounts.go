package mfatlas

import "context"

type CreateInvestmentAccountRequest struct {
	InvestorID string `json:"investor_id"`
	Code       string `json:"code"` // 2-char
	Name       string `json:"name"`
}

type InvestmentAccount struct {
	ID         string `json:"id"`
	InvestorID string `json:"investor_id"`
	Code       string `json:"code"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at"`
}

func (c *Client) CreateInvestmentAccount(ctx context.Context, req CreateInvestmentAccountRequest) (*InvestmentAccount, error) {
	var out InvestmentAccount
	if err := c.do(ctx, "POST", "/api/investment-accounts/v1/", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetInvestmentAccount(ctx context.Context, id string) (*InvestmentAccount, error) {
	var out InvestmentAccount
	if err := c.do(ctx, "GET", "/api/investment-accounts/v1/"+id, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListInvestmentAccounts(ctx context.Context, investorID string) ([]InvestmentAccount, error) {
	var out ListEnvelope[InvestmentAccount]
	opts := []RequestOption{}
	if investorID != "" {
		opts = append(opts, WithQuery(map[string]string{"investor_id": investorID}))
	}
	if err := c.do(ctx, "GET", "/api/investment-accounts/v1/", nil, &out, opts...); err != nil {
		return nil, err
	}
	return out.Data, nil
}
