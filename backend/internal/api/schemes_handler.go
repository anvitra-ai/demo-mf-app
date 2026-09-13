package api

import "net/http"

func (a *App) handleListSchemes(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	limit := r.URL.Query().Get("limit")
	if limit == "" {
		limit = "20"
	}
	schemes, err := a.MF.ListSchemes(r.Context(), search, limit)
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
