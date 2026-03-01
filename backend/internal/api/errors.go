package api

import (
	"errors"
	"net/http"

	"webcrawler/internal/crawler"
	"webcrawler/internal/util"
)

type apiErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

func writeAPIError(w http.ResponseWriter, status int, code, message string, details any) {
	util.WriteJSON(w, status, map[string]any{
		"error": apiErrorPayload{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

func writeValidationError(w http.ResponseWriter, message string, details any) {
	writeAPIError(w, http.StatusBadRequest, "validation_error", message, details)
}

func writeTargetPolicyError(w http.ResponseWriter, err error) {
	var targetErr *crawler.TargetPolicyError
	if errors.As(err, &targetErr) {
		writeAPIError(w, http.StatusBadRequest, targetErr.Code, targetErr.Message, nil)
		return
	}
	writeAPIError(w, http.StatusBadRequest, "target_not_allowed", "target URL is not allowed", nil)
}
