package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"demo-mf-app/internal/mfatlas"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		if err := json.NewEncoder(w).Encode(v); err != nil {
			log.Printf("writeJSON encode error: %v", err)
		}
	}
}

func readJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

// writeError renders both our own upstream *mfatlas.APIError and generic
// errors as a consistent {error:{code,message}} body so the frontend can
// branch on `code` exactly like the guide's §6 recommends server-side.
func writeError(w http.ResponseWriter, err error) {
	var apiErr *mfatlas.APIError
	if errors.As(err, &apiErr) {
		status := apiErr.HTTPStatus
		if status == 0 {
			status = http.StatusBadGateway
		}
		writeJSON(w, status, map[string]any{
			"error": map[string]any{
				"code":            apiErr.Code,
				"message":         apiErr.Message,
				"provider_remark": apiErr.ProviderRemark,
				"details":         apiErr.Details,
			},
		})
		return
	}

	log.Printf("internal error: %v", err)
	writeJSON(w, http.StatusInternalServerError, map[string]any{
		"error": map[string]any{
			"code":    "INTERNAL",
			"message": err.Error(),
		},
	})
}

func writeNotFound(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusNotFound, map[string]any{
		"error": map[string]any{"code": "NOT_FOUND", "message": message},
	})
}

func writeBadRequest(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusBadRequest, map[string]any{
		"error": map[string]any{"code": "VALIDATION_FAILED", "message": message},
	})
}
