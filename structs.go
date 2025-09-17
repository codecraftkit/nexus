package nexus

import (
	"github.com/rs/cors"
	"net/http"
	"regexp"
)

// Server is a struct that contains the server's configuration and endpoints
type Server struct {
	ServerName           string
	ServerNumber         string
	RunningServerMessage string
	Secret               string
	Debug                bool
	Port                 string
	Middlewares          []func(next http.Handler, server *Server) http.Handler
	Endpoints            [][]Endpoint
	EndpointsPaths       map[string]*Endpoint
	CorsOptions          cors.Options
	//SecretMiddleware     func(next http.Handler, server *Server) http.Handler // Replace the default secret middleware
	Settings *Settings
}

type Settings struct {
	IgnoreSecret bool
	PathPrefix   string
}

// Endpoint is a struct that contains the endpoint's configuration and handlers
type Endpoint struct {
	Path              string
	HandlerFunc       http.HandlerFunc
	Handler           http.Handler                          // Handler is a http.Handler and is used to create a new http.Handler with the server's middlewares and endpoints
	HandlerServerFunc func(server *Server) http.HandlerFunc // HandlerServerFunc is a function that returns a http.HandlerFunc and is used to create a new http.HandlerFunc with the server's middlewares and endpoints
	Options           EndpointOptions
	RegexPattern      *regexp.Regexp
}

// EndpointOptions is a struct that contains the endpoint's options
type EndpointOptions struct {
	IsPublic                 bool
	NoRequiresAuthentication bool
}
