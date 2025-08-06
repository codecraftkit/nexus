package nexus

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
)

type ResponsePagination struct {
	TotalPages   int64       `json:"total_pages"`
	Total        int64       `json:"total"`
	CurrentPage  int64       `json:"current_page"`
	From         int64       `json:"from"`
	To           int64       `json:"to"`
	Offset       int64       `json:"offset"`
	Limit        int64       `json:"limit"`
	PerPage      int         `json:"per_page"`
	Path         string      `json:"path"`
	FirstPageURL string      `json:"first_page_url"`
	NextPageURL  string      `json:"next_page_url"`
	LastPageURL  string      `json:"last_page_url"`
	PrevPageURL  string      `json:"prev_page_url"`
	Data         interface{} `json:"data"`
}

type PaginationOptions struct {
	Page    int64
	Limit   int64
	Skip    int64
	Payload interface{}
	Path    string
	Total   int64
}

type ErrorResponse struct {
	Code     int               `json:"code"`
	Message  string            `json:"message"`
	CodeName string            `json:"code_name"`
	Errors   map[string]string `json:"errors"`
}

// ResponseWithJSON return a json response
func ResponseWithJSON(w http.ResponseWriter, code int, payload interface{}) error {
	response, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
	return nil
}

// ResponseWithError return a json response with an error
func ResponseWithError(w http.ResponseWriter, code int, msg string) error {
	return ResponseWithJSON(w, code, map[string]string{"error": msg})
}

func ResponseJsonWithError(w http.ResponseWriter, code int, errors interface{}, errorResponse *ErrorResponse) error {
	if errorResponse == nil {
		errorResponse = &ErrorResponse{
			Code:     99999,
			Message:  "Internal Server Error",
			CodeName: "internal_server_error",
			Errors:   errors.(map[string]string),
		}
	}
	return ResponseWithJSON(w, code, errorResponse)
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

	w.WriteHeader(code)
	w.Write(response)
	return nil
}

func ResponseWithPaginationTest(w http.ResponseWriter, code int, options *PaginationOptions) error {

	paginated := ResponsePagination{
		Data:        options.Payload,
		Limit:       options.Limit,
		Offset:      options.Skip,
		CurrentPage: options.Page,
		Path:        options.Path,
		Total:       options.Total,
	}

	paginated.Set()

	return ResponseWithJSON(w, code, paginated)
}

func GetOptions(opt url.Values) (skip, limit, page int64) {
	page = int64(1)
	limit = int64(20)

	if opt.Get("page") != "" {
		p, pageErr := strconv.ParseInt(opt.Get("page"), 10, 64)
		if pageErr == nil {
			page = p
		}
	}

	if opt.Get("limit") != "" {
		l, limitErr := strconv.ParseInt(opt.Get("limit"), 10, 64)
		if limitErr == nil {
			limit = l
		}
	}

	skip = (page - 1) * limit

	return
}

func (p *ResponsePagination) getLenOfData() (int, error) {
	// Usamos la reflexión para inspeccionar el tipo de la interfaz Data
	value := reflect.ValueOf(p.Data)

	// Verificamos si el tipo de Data es un slice (arreglo dinámico)
	if value.Kind() == reflect.Slice {
		fmt.Println("value")
		// Si es un slice, obtenemos su longitud
		return value.Len(), nil
	}

	// Si Data no es un slice, devolvemos un error
	return 0, fmt.Errorf("el campo Data no es un slice, es de tipo: %s", value.Kind())
}

func (p *ResponsePagination) Set() {
	totalPages := int64(math.Ceil(float64(p.Total) / float64(p.Limit)))

	if p.CurrentPage > totalPages {
		p = &ResponsePagination{}
	}

	pageUrlFormat := "%s?page=%d&limit=%d"
	lenData, _ := p.getLenOfData()

	p.PerPage = lenData
	p.TotalPages = totalPages
	p.To = p.Offset + int64(lenData)

	p.From = 1
	p.FirstPageURL = ""
	p.PrevPageURL = ""
	if p.CurrentPage > 1 {
		p.From = p.Offset + 1
		p.FirstPageURL = fmt.Sprintf(pageUrlFormat, p.Path, 1, p.Limit)
		p.PrevPageURL = fmt.Sprintf(pageUrlFormat, p.Path, p.CurrentPage-1, p.Limit)
	}

	p.LastPageURL = ""
	p.NextPageURL = ""
	if p.CurrentPage < totalPages {
		p.LastPageURL = fmt.Sprintf(pageUrlFormat, p.Path, totalPages, p.Limit)
		p.NextPageURL = fmt.Sprintf(pageUrlFormat, p.Path, p.CurrentPage+1, p.Limit)
	}

}
