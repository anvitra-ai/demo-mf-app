package mfatlas

import "context"

type ChequeDetails struct {
	Number string `json:"number"`
	Date   string `json:"date"`
}

type CreatePaymentRequest struct {
	InvestorID          string         `json:"investor_id"`
	InvestmentAccountID string         `json:"investment_account_id,omitempty"`
	OrderIDs            []string       `json:"order_ids"`
	Mode                string         `json:"mode"` // MANDATE|CHEQUE|UPI|NETBANKING|NEFT_RTGS
	MandateID           string         `json:"mandate_id,omitempty"`
	BankAccountID       string         `json:"bank_account_id,omitempty"`
	VPA                 string         `json:"vpa,omitempty"`
	Cheque              *ChequeDetails `json:"cheque,omitempty"`
}

type Payment struct {
	ID                  string   `json:"id"`
	InvestorID          string   `json:"investor_id"`
	InvestmentAccountID string   `json:"investment_account_id"`
	Mode                string   `json:"mode"`
	Status              string   `json:"status"`
	Amount              float64  `json:"amount"`
	BasketID            string   `json:"basket_id"`
	PaymentURL          string   `json:"payment_url"`
	OrderIDs            []string `json:"order_ids"`
	CreatedAt           string   `json:"created_at"`
	UpdatedAt           string   `json:"updated_at"`
}

type SubmitUTRRequest struct {
	UTR          string `json:"utr"`
	TransferDate string `json:"transfer_date"`
	BankName     string `json:"bank_name"`
	IFSC         string `json:"ifsc"`
	AccountNo    string `json:"account_no"`
}

func (c *Client) CreatePayment(ctx context.Context, req CreatePaymentRequest, idempotencyKey string) (*Payment, error) {
	var out Payment
	if err := c.do(ctx, "POST", "/api/payments/v1/", req, &out, WithIdempotencyKey(idempotencyKey)); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListPayments(ctx context.Context, investorID string) ([]Payment, error) {
	var out []Payment
	opts := []RequestOption{}
	if investorID != "" {
		opts = append(opts, WithQuery(map[string]string{"investor_id": investorID}))
	}
	if err := c.do(ctx, "GET", "/api/payments/v1/", nil, &out, opts...); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetPayment(ctx context.Context, id string) (*Payment, error) {
	var out Payment
	if err := c.do(ctx, "GET", "/api/payments/v1/"+id, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) SubmitUTR(ctx context.Context, paymentID string, req SubmitUTRRequest) error {
	return c.do(ctx, "POST", "/api/payments/v1/"+paymentID+"/utr", req, nil)
}
