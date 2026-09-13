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

// SchemeFilters are the optional query params GET /api/schemes/v1/ accepts
// for narrowing down the master list (per the OpenAPI spec: amc, search,
// plan, option, category, purchase_allowed, sip_allowed, limit, cursor).
type SchemeFilters struct {
	Search          string
	AMC             string
	Category        string
	Plan            string
	Option          string
	PurchaseAllowed string // "true"/"false", left empty for no filter
	SIPAllowed      string
	Limit           string
}

func (c *Client) ListSchemes(ctx context.Context, f SchemeFilters) ([]Scheme, error) {
	var out []Scheme
	q := map[string]string{}
	if f.Search != "" {
		q["search"] = f.Search
	}
	if f.AMC != "" {
		q["amc"] = f.AMC
	}
	if f.Category != "" {
		q["category"] = f.Category
	}
	if f.Plan != "" {
		q["plan"] = f.Plan
	}
	if f.Option != "" {
		q["option"] = f.Option
	}
	if f.PurchaseAllowed != "" {
		q["purchase_allowed"] = f.PurchaseAllowed
	}
	if f.SIPAllowed != "" {
		q["sip_allowed"] = f.SIPAllowed
	}
	if f.Limit != "" {
		q["limit"] = f.Limit
	}
	if err := c.do(ctx, "GET", "/api/schemes/v1/", nil, &out, WithQuery(q)); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetScheme(ctx context.Context, code string) (*Scheme, error) {
	var out Scheme
	if err := c.do(ctx, "GET", "/api/schemes/v1/"+code, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
