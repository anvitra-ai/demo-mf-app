package mfatlas

import "context"

type PrimaryHolder struct {
	Name   string `json:"name"`
	PAN    string `json:"pan"`
	DOB    string `json:"dob"`
	Mobile string `json:"mobile"`
	Email  string `json:"email"`
	Gender string `json:"gender,omitempty"`
}

type Address struct {
	Line1   string `json:"line1"`
	Line2   string `json:"line2,omitempty"`
	City    string `json:"city"`
	State   string `json:"state"`
	Pincode string `json:"pincode"`
	Country string `json:"country"`
}

type Contact struct {
	Email  string `json:"email"`
	Mobile string `json:"mobile"`
}

type BankAccountInput struct {
	AccountNo   string `json:"account_no"`
	AccountType string `json:"account_type"` // SB|CB|NE|NO
	IFSC        string `json:"ifsc"`
	IsDefault   bool   `json:"is_default"`
}

type CreateInvestorRequest struct {
	PrimaryHolder PrimaryHolder      `json:"primary_holder"`
	Address       Address            `json:"address"`
	Contact       Contact            `json:"contact"`
	BankAccounts  []BankAccountInput `json:"bank_accounts"`
	HoldingType   string             `json:"holding_type"` // SINGLE|JOINT|ANYONE_OR_SURVIVOR
	TaxStatus     string             `json:"tax_status"`
	ClientRef     string             `json:"client_ref,omitempty"`
	Submit        bool               `json:"submit"`
}

type ProviderStep struct {
	Step      string `json:"step"`
	Status    string `json:"status"`
	Remark    string `json:"remark"`
	Timestamp string `json:"timestamp"`
}

type InvestorProvider struct {
	Type       string `json:"type"`
	ClientCode string `json:"client_code"`
	Status     string `json:"status"`
}

type Investor struct {
	ID            string             `json:"id"`
	Status        string             `json:"status"`
	Provider      string             `json:"provider"`
	Providers     []InvestorProvider `json:"providers"`
	AuthLink      string             `json:"auth_link"`
	ClientRef     string             `json:"client_ref"`
	CreatedAt     string             `json:"created_at"`
	UpdatedAt     string             `json:"updated_at"`
	ProviderSteps []ProviderStep     `json:"provider_steps"`
	PrimaryHolder *PrimaryHolder     `json:"primary_holder"`
}

type InvestorBankAccount struct {
	ID           string `json:"id"`
	AccountNo    string `json:"account_no"`
	AccountType  string `json:"account_type"`
	IFSC         string `json:"ifsc"`
	IsDefault    bool   `json:"is_default"`
	Status       string `json:"status"`
	StatusRemark string `json:"status_remark"`
}

type SyncResult struct {
	Checked                int `json:"checked"`
	Reconciled             int `json:"reconciled"`
	NotFoundAtNSE          int `json:"not_found_at_nse"`
	SkippedAtNSE           int `json:"skipped_at_nse"`
	BankAccountsChecked    int `json:"bank_accounts_checked"`
	BankAccountsReconciled int `json:"bank_accounts_reconciled"`
}

func (c *Client) CreateInvestor(ctx context.Context, req CreateInvestorRequest) (*Investor, error) {
	var out Investor
	if err := c.do(ctx, "POST", "/api/investors/v1/", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListInvestors(ctx context.Context) ([]Investor, error) {
	var out []Investor
	if err := c.do(ctx, "GET", "/api/investors/v1/", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetInvestor(ctx context.Context, id string, withAuthLink bool) (*Investor, error) {
	var out Investor
	opts := []RequestOption{}
	if withAuthLink {
		opts = append(opts, WithQuery(map[string]string{"auth_link": "true"}))
	}
	if err := c.do(ctx, "GET", "/api/investors/v1/"+id, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) PatchInvestor(ctx context.Context, id string, req CreateInvestorRequest) (*Investor, error) {
	var out Investor
	if err := c.do(ctx, "PATCH", "/api/investors/v1/"+id, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) RetryInvestor(ctx context.Context, id string) (*Investor, error) {
	var out Investor
	if err := c.do(ctx, "POST", "/api/investors/v1/"+id+"/retry", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListInvestorBankAccounts(ctx context.Context, investorID string) ([]InvestorBankAccount, error) {
	var out []InvestorBankAccount
	if err := c.do(ctx, "GET", "/api/investors/v1/"+investorID+"/bank-accounts", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) SyncInvestors(ctx context.Context) (*SyncResult, error) {
	var out SyncResult
	if err := c.do(ctx, "POST", "/api/investors/v1/sync", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
