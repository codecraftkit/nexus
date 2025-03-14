package nexus

import (
	"encoding/json"
	"net/http"
)

// ResponseWithJSON return a json response
func ResponseWithJSON(w http.ResponseWriter, code int, payload interface{}) error {
	response, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(code)
	w.Write(response)
	return nil
}

// ResponseWithError return a json response with an error
func ResponseWithError(w http.ResponseWriter, code int, msg string) error {
	return ResponseWithJSON(w, code, map[string]string{"error": msg})
}
