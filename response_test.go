package nexus

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// --- ResponseWithJSON ---

func TestResponseWithJSON_StatusAndContentType(t *testing.T) {
	w := httptest.NewRecorder()
	payload := map[string]string{"key": "value"}
	err := ResponseWithJSON(w, http.StatusOK, payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json, got %s", ct)
	}

	var result map[string]string
	json.Unmarshal(w.Body.Bytes(), &result)
	if result["key"] != "value" {
		t.Fatalf("expected value, got %s", result["key"])
	}
}

func TestResponseWithJSON_Code0DefaultsTo500(t *testing.T) {
	w := httptest.NewRecorder()
	err := ResponseWithJSON(w, 0, "error")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestResponseWithJSON_DifferentStatusCodes(t *testing.T) {
	codes := []int{http.StatusCreated, http.StatusBadRequest, http.StatusNotFound}
	for _, code := range codes {
		w := httptest.NewRecorder()
		ResponseWithJSON(w, code, nil)
		if w.Code != code {
			t.Fatalf("expected %d, got %d", code, w.Code)
		}
	}
}

// --- ResponseWithError ---

func TestResponseWithError(t *testing.T) {
	w := httptest.NewRecorder()
	err := ResponseWithError(w, http.StatusBadRequest, "bad input")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	var resp ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected code 400 in body, got %d", resp.Code)
	}
	if resp.Message != "bad input" {
		t.Fatalf("expected 'bad input', got %s", resp.Message)
	}
	if resp.CodeName != "internal_server_error" {
		t.Fatalf("expected 'internal_server_error', got %s", resp.CodeName)
	}
	if resp.Errors["error"] != "bad input" {
		t.Fatalf("expected error map entry, got %v", resp.Errors)
	}
}

func TestResponseWithError_Code0DefaultsTo500(t *testing.T) {
	w := httptest.NewRecorder()
	ResponseWithError(w, 0, "fail")
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// --- ResponseJsonWithError ---

func TestResponseJsonWithError_NilFallback(t *testing.T) {
	w := httptest.NewRecorder()
	err := ResponseJsonWithError(w, http.StatusInternalServerError, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var resp ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != 99999 {
		t.Fatalf("expected code 99999, got %d", resp.Code)
	}
	if resp.Message != "Internal Server Error" {
		t.Fatalf("expected 'Internal Server Error', got %s", resp.Message)
	}
}

func TestResponseJsonWithError_EOFConversion(t *testing.T) {
	w := httptest.NewRecorder()
	errResp := &ErrorResponse{
		Code:     400,
		Message:  "EOF",
		CodeName: "bad_request",
	}
	ResponseJsonWithError(w, http.StatusBadRequest, errResp)

	var resp ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Message != "BODY_REQUIRED" {
		t.Fatalf("expected 'BODY_REQUIRED', got %s", resp.Message)
	}
}

func TestResponseJsonWithError_Code0DefaultsTo500(t *testing.T) {
	w := httptest.NewRecorder()
	ResponseJsonWithError(w, 0, nil)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// --- GetOptions ---

func TestGetOptions_Defaults(t *testing.T) {
	vals := url.Values{}
	skip, limit, page := GetOptions(vals)
	if page != 1 {
		t.Fatalf("expected page 1, got %d", page)
	}
	if limit != 20 {
		t.Fatalf("expected limit 20, got %d", limit)
	}
	if skip != 0 {
		t.Fatalf("expected skip 0, got %d", skip)
	}
}

func TestGetOptions_CustomValues(t *testing.T) {
	vals := url.Values{"page": {"3"}, "limit": {"10"}}
	skip, limit, page := GetOptions(vals)
	if page != 3 {
		t.Fatalf("expected page 3, got %d", page)
	}
	if limit != 10 {
		t.Fatalf("expected limit 10, got %d", limit)
	}
	if skip != 20 { // (3-1)*10
		t.Fatalf("expected skip 20, got %d", skip)
	}
}

func TestGetOptions_InvalidValues(t *testing.T) {
	vals := url.Values{"page": {"abc"}, "limit": {"xyz"}}
	skip, limit, page := GetOptions(vals)
	// Invalid values should fall back to defaults
	if page != 1 {
		t.Fatalf("expected page 1 on invalid, got %d", page)
	}
	if limit != 20 {
		t.Fatalf("expected limit 20 on invalid, got %d", limit)
	}
	if skip != 0 {
		t.Fatalf("expected skip 0, got %d", skip)
	}
}

// --- ResponsePagination.Set ---

func TestResponsePaginationSet_FirstPage(t *testing.T) {
	p := &ResponsePagination{
		Total:       50,
		Limit:       10,
		CurrentPage: 1,
		Offset:      0,
		Path:        "/api/items",
		Data:        []string{"a", "b", "c"},
	}
	p.Set()

	if p.TotalPages != 5 {
		t.Fatalf("expected 5 total pages, got %d", p.TotalPages)
	}
	if p.PerPage != 3 {
		t.Fatalf("expected 3 per page, got %d", p.PerPage)
	}
	if p.From != 1 {
		t.Fatalf("expected from 1, got %d", p.From)
	}
	if p.To != 3 {
		t.Fatalf("expected to 3, got %d", p.To)
	}
	// First page: no prev/first URLs
	if p.FirstPageURL != "" {
		t.Fatalf("expected empty first page URL on page 1, got %s", p.FirstPageURL)
	}
	if p.PrevPageURL != "" {
		t.Fatalf("expected empty prev URL on page 1, got %s", p.PrevPageURL)
	}
	// Has next/last
	if p.NextPageURL == "" {
		t.Fatal("expected next page URL")
	}
	if p.LastPageURL == "" {
		t.Fatal("expected last page URL")
	}
}

func TestResponsePaginationSet_MiddlePage(t *testing.T) {
	p := &ResponsePagination{
		Total:       50,
		Limit:       10,
		CurrentPage: 3,
		Offset:      20,
		Path:        "/api/items",
		Data:        []string{"a", "b"},
	}
	p.Set()

	if p.From != 21 {
		t.Fatalf("expected from 21, got %d", p.From)
	}
	if p.To != 22 {
		t.Fatalf("expected to 22, got %d", p.To)
	}
	if p.FirstPageURL == "" {
		t.Fatal("expected first page URL on middle page")
	}
	if p.PrevPageURL == "" {
		t.Fatal("expected prev page URL on middle page")
	}
	if p.NextPageURL == "" {
		t.Fatal("expected next page URL on middle page")
	}
	if p.LastPageURL == "" {
		t.Fatal("expected last page URL on middle page")
	}
}

func TestResponsePaginationSet_LastPage(t *testing.T) {
	p := &ResponsePagination{
		Total:       50,
		Limit:       10,
		CurrentPage: 5,
		Offset:      40,
		Path:        "/api/items",
		Data:        []string{"a"},
	}
	p.Set()

	// Last page: no next/last URLs
	if p.NextPageURL != "" {
		t.Fatalf("expected empty next URL on last page, got %s", p.NextPageURL)
	}
	if p.LastPageURL != "" {
		t.Fatalf("expected empty last URL on last page, got %s", p.LastPageURL)
	}
	// Has first/prev
	if p.FirstPageURL == "" {
		t.Fatal("expected first page URL on last page")
	}
	if p.PrevPageURL == "" {
		t.Fatal("expected prev page URL on last page")
	}
}

func TestResponsePaginationSet_URLFormat(t *testing.T) {
	p := &ResponsePagination{
		Total:       30,
		Limit:       10,
		CurrentPage: 2,
		Offset:      10,
		Path:        "/items",
		Data:        []string{"a"},
	}
	p.Set()

	expected := "/items?page=1&limit=10"
	if p.FirstPageURL != expected {
		t.Fatalf("expected %s, got %s", expected, p.FirstPageURL)
	}
	expected = "/items?page=1&limit=10"
	if p.PrevPageURL != expected {
		t.Fatalf("expected %s, got %s", expected, p.PrevPageURL)
	}
	expected = "/items?page=3&limit=10"
	if p.NextPageURL != expected {
		t.Fatalf("expected %s, got %s", expected, p.NextPageURL)
	}
	expected = "/items?page=3&limit=10"
	if p.LastPageURL != expected {
		t.Fatalf("expected %s, got %s", expected, p.LastPageURL)
	}
}
