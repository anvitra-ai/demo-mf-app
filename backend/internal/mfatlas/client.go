// Package mfatlas is a thin client for the mf-atlas.space Mutual Fund API.
// See MF-API-INTEGRATION-GUIDE.md at the repo root for the flows this wraps.
package mfatlas

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// APIError is the decoded error envelope from mf-atlas. Callers must branch
// on Code, never on Message/ProviderRemark (see guide §6).
type APIError struct {
	HTTPStatus     int
	Code           string
	Message        string
	Details        []map[string]any
	ProviderRemark string
	RequestID      string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("mf-atlas error %s (http %d): %s", e.Code, e.HTTPStatus, e.Message)
}

// IsRetryable reports whether the guide's retry policy (§6) says this error
// is safe to retry with backoff.
func (e *APIError) IsRetryable() bool {
	return e.Code == "PROVIDER_UNAVAILABLE" || e.Code == "RATE_LIMITED"
}

type envelope struct {
	Success    bool            `json:"success"`
	Data       json.RawMessage `json:"data"`
	NextCursor string          `json:"next_cursor"`
	Message    string          `json:"message"`
	Error      *struct {
		Code           string           `json:"code"`
		Message        string           `json:"message"`
		Details        []map[string]any `json:"details"`
		ProviderRemark string           `json:"provider_remark"`
		RequestID      string           `json:"request_id"`
	} `json:"error"`
}

// Client is a bearer-token-authenticated HTTP client for mf-atlas.
type Client struct {
	baseURL      string
	clientID     string
	clientSecret string
	http         *http.Client

	mu          sync.Mutex
	accessToken string
	expiresAt   time.Time
}

func NewClient(baseURL, clientID, clientSecret string) *Client {
	return &Client{
		baseURL:      strings.TrimRight(baseURL, "/"),
		clientID:     clientID,
		clientSecret: clientSecret,
		http:         &http.Client{Timeout: 30 * time.Second},
	}
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

// ensureToken returns a valid bearer token, fetching a new one if absent or
// close to expiry. No refresh token exists in this API — see guide §0.
func (c *Client) ensureToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.accessToken != "" && time.Now().Before(c.expiresAt.Add(-60*time.Second)) {
		return c.accessToken, nil
	}

	if c.clientID == "" || c.clientSecret == "" {
		return "", &APIError{Code: "NOT_CONFIGURED", Message: "MF_ATLAS_CLIENT_ID/MF_ATLAS_CLIENT_SECRET are not set"}
	}

	body, _ := json.Marshal(map[string]string{
		"client_id":     c.clientID,
		"client_secret": c.clientSecret,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/auth/v1/token", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode >= 400 {
		return "", parseAPIError(resp.StatusCode, raw)
	}

	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return "", fmt.Errorf("decoding token response: %w", err)
	}
	var tok tokenResponse
	if err := json.Unmarshal(env.Data, &tok); err != nil {
		return "", fmt.Errorf("decoding token data: %w", err)
	}

	c.accessToken = tok.AccessToken
	c.expiresAt = time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
	return c.accessToken, nil
}

func parseAPIError(status int, raw []byte) *APIError {
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil || env.Error == nil {
		return &APIError{HTTPStatus: status, Code: "UNKNOWN", Message: string(raw)}
	}
	return &APIError{
		HTTPStatus:     status,
		Code:           env.Error.Code,
		Message:        env.Error.Message,
		Details:        env.Error.Details,
		ProviderRemark: env.Error.ProviderRemark,
		RequestID:      env.Error.RequestID,
	}
}

// requestOpts customizes a single call.
type requestOpts struct {
	idempotencyKey string
	query          map[string]string
}

// RequestOption customizes a do() call.
type RequestOption func(*requestOpts)

// WithIdempotencyKey attaches an Idempotency-Key header, required in
// practice on POST /api/orders/v1/ (guide §0, §4.3).
func WithIdempotencyKey(key string) RequestOption {
	return func(o *requestOpts) { o.idempotencyKey = key }
}

// WithQuery adds query-string parameters.
func WithQuery(q map[string]string) RequestOption {
	return func(o *requestOpts) { o.query = q }
}

// do performs an authenticated JSON call against mf-atlas and decodes
// `data` into out (if non-nil). It surfaces upstream errors as *APIError.
func (c *Client) do(ctx context.Context, method, path string, body any, out any, opts ...RequestOption) error {
	o := &requestOpts{}
	for _, opt := range opts {
		opt(o)
	}

	token, err := c.ensureToken(ctx)
	if err != nil {
		return err
	}

	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(b)
	}

	url := c.baseURL + path
	if len(o.query) > 0 {
		q := make([]string, 0, len(o.query))
		for k, v := range o.query {
			if v != "" {
				q = append(q, k+"="+v)
			}
		}
		if len(q) > 0 {
			url += "?" + strings.Join(q, "&")
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if o.idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", o.idempotencyKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		return parseAPIError(resp.StatusCode, raw)
	}

	if out == nil {
		return nil
	}

	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return fmt.Errorf("decoding response from %s %s: %w", method, path, err)
	}
	if len(env.Data) == 0 {
		return nil
	}
	if err := json.Unmarshal(env.Data, out); err != nil {
		return fmt.Errorf("decoding data from %s %s: %w", method, path, err)
	}
	return nil
}
