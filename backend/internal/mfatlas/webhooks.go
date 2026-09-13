package mfatlas

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

type CreateWebhookSubscriptionRequest struct {
	URL    string   `json:"url"`
	Events []string `json:"events"`
}

type WebhookSubscription struct {
	ID     string   `json:"id"`
	URL    string   `json:"url"`
	Events []string `json:"events"`
	Secret string   `json:"secret"` // shown once, at creation only
}

func (c *Client) CreateWebhookSubscription(ctx context.Context, req CreateWebhookSubscriptionRequest) (*WebhookSubscription, error) {
	var out WebhookSubscription
	if err := c.do(ctx, "POST", "/api/webhooks/v1/subscriptions", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// VerifySignature checks the `X-MF-Signature` header of the form
// "t=<unix>,v1=<hex hmac-sha256>" against the raw request body, per guide §5.3.
func VerifySignature(header, rawBody, secret string) bool {
	parts := strings.Split(header, ",")
	if len(parts) != 2 {
		return false
	}
	var t, sig string
	for _, p := range parts {
		kv := strings.SplitN(p, "=", 2)
		if len(kv) != 2 {
			return false
		}
		switch kv[0] {
		case "t":
			t = kv[1]
		case "v1":
			sig = kv[1]
		}
	}
	if t == "" || sig == "" {
		return false
	}
	if _, err := strconv.ParseInt(t, 10, 64); err != nil {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(fmt.Sprintf("%s.%s", t, rawBody)))
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(sig))
}
