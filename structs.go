package nexus

import (
	"net/http"
	"regexp"
	"time"

	"github.com/rs/cors"
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
	Settings             *Settings
}

type Settings struct {
	IgnoreSecret bool
	PathPrefix   string

	// WriteTimeout and ReadTimeout override the server defaults
	// (DefaultWriteTimeout / DefaultReadTimeout). Zero means "not set" and keeps
	// the default, so existing servers behave exactly as before.
	//
	// They were hardcoded at 15s until now. That is a sane default for a REST
	// API and impossible for anything long-lived: an SSE stream, a chunked
	// download, or an upload over a slow connection dies mid-flight, and the
	// client sees a truncated body rather than an error.
	//
	// # Do NOT reach for a zero timeout to stream
	//
	// Zero here means "keep the default", not "no timeout" — deliberately.
	// Disabling the write timeout for the WHOLE server so that one endpoint can
	// stream lets a single stuck client hold a connection forever, and takes the
	// protection away from every other route to fix one.
	//
	// The right tool is per-connection and lives in the standard library:
	//
	//	// inside a streaming handler
	//	rc := http.NewResponseController(w)
	//	if err := rc.SetWriteDeadline(time.Time{}); err != nil {
	//		// the server does not support deadline control; fail loudly
	//	}
	//
	// That clears the deadline for THAT connection and leaves the server's
	// default protecting everything else. These fields are for when the whole
	// service needs a different number — a gateway that proxies slow upstreams,
	// a service whose every route uploads.
	WriteTimeout time.Duration
	ReadTimeout  time.Duration
}

// Server timeout defaults, applied when Settings leaves them at zero. They are
// the values Run() hardcoded before they were configurable, so upgrading
// changes nothing on its own.
const (
	DefaultWriteTimeout = 15 * time.Second
	DefaultReadTimeout  = 15 * time.Second
)

// writeTimeout resolves the configured value against the default.
//
// Split out of Run() so it can be tested: Run() ends in ListenAndServe and
// log.Fatal, so anything decided inline there is only reachable by starting a
// real server on a real port.
func (settings *Settings) writeTimeout() time.Duration {
	if settings == nil || settings.WriteTimeout == 0 {
		return DefaultWriteTimeout
	}

	return settings.WriteTimeout
}

// readTimeout resolves the configured value against the default.
func (settings *Settings) readTimeout() time.Duration {
	if settings == nil || settings.ReadTimeout == 0 {
		return DefaultReadTimeout
	}

	return settings.ReadTimeout
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
	IgnorePrefix             bool
}

type GroupOptions struct {
	Middlewares []func(next http.Handler) http.Handler
}
