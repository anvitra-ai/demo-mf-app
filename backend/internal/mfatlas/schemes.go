package mfatlas

import "context"

type Scheme struct {
	SchemeCode         string  `json:"scheme_code"`
	ISIN               string  `json:"isin"`
	Name               string  `json:"name"`
	AMCCode            string  `json:"amc_code"`
	AMCName            string  `json:"amc_name"`
	Category           string  `json:"category"`
	Plan               string  `json:"plan"`
	Option             string  `json:"option"`
	NAV                float64 `json:"nav"`
	MinPurchase        float64 `json:"min_purchase"`
	MaxPurchase        float64 `json:"max_purchase"`
	MinAdditional      float64 `json:"min_additional"`
	PurchaseMultiple   float64 `json:"purchase_multiple"`
	MinSIP             float64 `json:"min_sip"`
	MinRedemptionUnits float64 `json:"min_redemption_units"`
	PurchaseAllowed    bool    `json:"purchase_allowed"`
	RedemptionAllowed  bool    `json:"redemption_allowed"`
	SIPAllowed         bool    `json:"sip_allowed"`
	SwitchAllowed      bool    `json:"switch_allowed"`
	SettlementType     string  `json:"settlement_type"`
	LockInDays         int     `json:"lock_in_days"`
	ExitLoad           string  `json:"exit_load"`
	UpdatedAt          string  `json:"updated_at"`
}

func (c *Client) ListSchemes(ctx context.Context, search string, limit string) ([]Scheme, error) {
	var out ListEnvelope[Scheme]
	q := map[string]string{}
	if search != "" {
		q["search"] = search
	}
	if limit != "" {
		q["limit"] = limit
	}
	if err := c.do(ctx, "GET", "/api/schemes/v1/", nil, &out, WithQuery(q)); err != nil {
		return nil, err
	}
	return out.Data, nil
}

func (c *Client) GetScheme(ctx context.Context, code string) (*Scheme, error) {
	var out Scheme
	if err := c.do(ctx, "GET", "/api/schemes/v1/"+code, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
