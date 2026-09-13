// Package api implements the demo app's own JSON API (consumed by the
// vanilla-JS frontend) plus the mf-atlas webhook receiver. Handlers translate
// between our simplified DTOs and the mfatlas client, and keep the local
// Mongo mirror in internal/store up to date.
package api

import (
	"net/http"

	"demo-mf-app/internal/config"
	"demo-mf-app/internal/mfatlas"
	"demo-mf-app/internal/store"
)

type App struct {
	Cfg   *config.Config
	MF    *mfatlas.Client
	Store *store.Store
}

func NewRouter(app *App, frontendDir string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", app.handleHealth)

	mux.HandleFunc("POST /api/investors", app.handleCreateInvestor)
	mux.HandleFunc("GET /api/investors", app.handleListInvestors)
	mux.HandleFunc("GET /api/investors/{id}", app.handleGetInvestor)
	mux.HandleFunc("POST /api/investors/{id}/refresh", app.handleRefreshInvestor)
	mux.HandleFunc("POST /api/investors/{id}/retry", app.handleRetryInvestor)
	mux.HandleFunc("POST /api/investors/{id}/resubmit", app.handleResubmitInvestor)
	mux.HandleFunc("POST /api/investors/sync", app.handleSyncInvestors)

	mux.HandleFunc("POST /api/accounts", app.handleCreateAccount)
	mux.HandleFunc("GET /api/accounts", app.handleListAccounts)

	mux.HandleFunc("POST /api/mandates", app.handleCreateMandate)
	mux.HandleFunc("GET /api/mandates", app.handleListMandates)
	mux.HandleFunc("GET /api/mandates/{id}", app.handleGetMandate)
	mux.HandleFunc("POST /api/mandates/{id}/refresh", app.handleRefreshMandate)

	mux.HandleFunc("GET /api/schemes", app.handleListSchemes)
	mux.HandleFunc("GET /api/schemes/{code}", app.handleGetScheme)

	mux.HandleFunc("POST /api/orders", app.handleCreateOrder)
	mux.HandleFunc("GET /api/orders", app.handleListOrders)
	mux.HandleFunc("GET /api/orders/{id}", app.handleGetOrder)
	mux.HandleFunc("POST /api/orders/{id}/refresh", app.handleRefreshOrder)
	mux.HandleFunc("GET /api/orders/{id}/events", app.handleGetOrderEvents)

	mux.HandleFunc("POST /api/payments", app.handleCreatePayment)
	mux.HandleFunc("GET /api/payments", app.handleListPayments)
	mux.HandleFunc("GET /api/payments/{id}", app.handleGetPayment)
	mux.HandleFunc("POST /api/payments/{id}/refresh", app.handleRefreshPayment)
	mux.HandleFunc("POST /api/payments/{id}/utr", app.handleSubmitUTR)

	mux.HandleFunc("POST /hooks/mf-atlas", app.handleWebhook)

	mux.Handle("/", http.FileServer(http.Dir(frontendDir)))

	return withLogging(mux)
}

func (a *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}
