package api

import (
	"net/http"

	"demo-mf-app/internal/mfatlas"
)

func (a *App) handleListSchemes(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := q.Get("limit")
	if limit == "" {
		limit = "20"
	}
	schemes, err := a.MF.ListSchemes(r.Context(), mfatlas.SchemeFilters{
		Search:          q.Get("search"),
		AMC:             q.Get("amc"),
		Category:        q.Get("category"),
		Plan:            q.Get("plan"),
		Option:          q.Get("option"),
		PurchaseAllowed: q.Get("purchase_allowed"),
		SIPAllowed:      q.Get("sip_allowed"),
		Limit:           limit,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, schemes)
}

func (a *App) handleGetScheme(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	scheme, err := a.MF.GetScheme(r.Context(), code)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, scheme)
}
