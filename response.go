package nexus

import (
	"encoding/json"
	"net/http"
)

type ResponsePagination struct {
	CurrentPage  int         `json:"current_page"`
	FirstPageURL string      `json:"first_page_url"`
	From         int         `json:"from"`
	NextPageURL  string      `json:"next_page_url"`
	Path         string      `json:"path"`
	PerPage      int         `json:"per_page"`
	PrevPageURL  string      `json:"prev_page_url"`
	To           int         `json:"to"`
	Data         interface{} `json:"data"`
}

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

func ResponseWithPagination(w http.ResponseWriter, code int, payload interface{}) error {
	paginated := ResponsePagination{
		Data: payload,
	}
	response, err := json.Marshal(paginated)
	if err != nil {
		return err
	}
	paginated.Data = payload
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(code)
	w.Write(response)
	return nil
}
