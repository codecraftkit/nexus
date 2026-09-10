package nexus

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

// EndpointFunc contains the endpoint's functions

// EndpointIsPublic evalue if the endpoint is public
func (server *Server) EndpointIsPublic(r *http.Request) bool {
	endpoint, ok := server.GetEndpoint(r)
	return ok && endpoint.Options.IsPublic
}

func (server *Server) NoRequiresAuthentication(r *http.Request) bool {
	endpoint, ok := server.GetEndpoint(r)
	return ok && endpoint.Options.NoRequiresAuthentication
}

// GetEndpoint evaluate if a path exists in the endpoints and return the endpoint and a bool if exists
func (server *Server) GetEndpoint(r *http.Request) (*Endpoint, bool) {
	route := r.URL.Path
	method := r.Method
	path := fmt.Sprintf("%s %s", method, route)
	endpoint := server.matchRoute(path)
	if endpoint != nil {
		return endpoint, true
	}
	return nil, false
}

// GetEndpoints return all endpoints
func (server *Server) GetEndpoints() map[string]*Endpoint {
	return server.EndpointsPaths
}

// registerEndpoint add a endpoint to the endpoint's map
func (server *Server) registerEndpoint(endpoint Endpoint) {
	// Convertir a regex
	compiledRegex := convertToRegex(endpoint.Path)
	endpoint.RegexPattern = compiledRegex
	server.EndpointsPaths[endpoint.Path] = &endpoint
}

// PrepareEndpoints applies the path prefix, registers every endpoint on a mux
// and fills the route index that EndpointIsPublic and NoRequiresAuthentication
// read. It returns that mux. Run calls it and then serves the result.
//
// It is exported so a caller can build the SAME wiring without listening.
// Until now the index was filled only inside Run, and that had a consequence
// worth spelling out: a test that assembled its own mux saw an EMPTY index, so
// matchRoute returned nil and every endpoint looked non-public. An assertion
// like "this group requires the API secret" then passed for a reason that had
// nothing to do with the group — it would have passed just the same with the
// group marked public. That is a green result that does not mean what it looks
// like, and the only way to avoid it without duplicating this route matching
// somewhere else is to let callers run this exact code.
//
// It is idempotent: calling it twice neither prefixes the paths twice nor adds
// the library endpoints twice.
func (server *Server) PrepareEndpoints() *http.ServeMux {

	if server.Settings == nil {
		server.Settings = &Settings{}
	}

	mux := http.NewServeMux()

	if !server.endpointsPrepared {
		// A COPY of the library endpoints, not the package-level slice itself.
		//
		// Today every entry in ServerEndpoints sets IgnorePrefix, so the
		// prefixing below never rewrites them and sharing the backing array
		// happens to be harmless. This copy is what keeps it harmless: Serve
		// runs several servers at once, each with its own PathPrefix, so a
		// library endpoint added WITHOUT IgnorePrefix would have one server's
		// prefix land on every other server's routes. Cheap here, and silent if
		// it ever happened.
		server.Endpoints = append(server.Endpoints, append([]Endpoint(nil), ServerEndpoints...))
	}

	for i, endpoints := range server.Endpoints {
		for j, endpoint := range endpoints {

			if !server.endpointsPrepared && !endpoint.Options.IgnorePrefix {
				endpoint.Path = strings.Replace(
					endpoint.Path,
					" /",
					fmt.Sprintf(" %s/", server.Settings.PathPrefix),
					-1,
				)
			}
			server.Endpoints[i][j] = endpoint

			// If an endpoint has both Handler and HandlerFunc the server going to crash
			if endpoint.HandlerFunc != nil && endpoint.Handler != nil {
				panic("Endpoint cannot have both HandlerFunc and Handler")
			}
			if endpoint.HandlerServerFunc != nil {
				mux.HandleFunc(endpoint.Path, endpoint.HandlerServerFunc(server))
			}
			if endpoint.HandlerFunc != nil {
				mux.HandleFunc(endpoint.Path, endpoint.HandlerFunc)
			}
			if endpoint.Handler != nil {
				mux.Handle(endpoint.Path, endpoint.Handler)
			}
		}

		server.setEndpoints(endpoints)
	}

	server.endpointsPrepared = true

	return mux
}

// setEndpoints add a list of endpoints to the endpoint's map
func (server *Server) setEndpoints(endpoints []Endpoint) {

	if server.EndpointsPaths == nil {
		server.EndpointsPaths = make(map[string]*Endpoint)
	}

	for _, endpoint := range endpoints {
		if server.Debug {
			fmt.Println(endpoint.Path)
		}
		server.registerEndpoint(endpoint)
	}
}

func convertToRegex(pattern string) *regexp.Regexp {

	// Buscar parámetros en la forma {param}
	re := regexp.MustCompile(`\{([^}]+)\}`)

	// Reemplazar {param} por regex y almacenar nombres de parámetros
	regexPattern := re.ReplaceAllStringFunc(pattern, func(match string) string {
		return `([^/]+)` // Grupo de captura para valores dinámicos
	})

	// Agregar inicio ^ y fin $ para coincidencia exacta
	fullRegex := "^" + regexPattern + "$"

	// Compilar regex final
	compiledRegex := regexp.MustCompile(fullRegex)

	return compiledRegex
}

func pathMatches(compiledRegex *regexp.Regexp, url string) bool {
	matches := compiledRegex.FindStringSubmatch(url)
	if matches == nil {
		return false
	}
	return true
}

func (server *Server) matchRoute(url string) *Endpoint {
	for _, ep := range server.EndpointsPaths {
		if ep.RegexPattern != nil && ep.RegexPattern.MatchString(url) {
			return ep
		}
	}
	return nil
}

func RequestScheme(r *http.Request) string {
	// 1) Encabezados estándar de proxies/CDN
	if xf := r.Header.Get("X-Forwarded-Proto"); xf != "" {
		return xf
	}
	// 2) RFC 7239: Forwarded: proto=https; host=...
	if fwd := r.Header.Get("Forwarded"); fwd != "" {
		// búsqueda simple
		parts := strings.Split(fwd, ";")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if strings.HasPrefix(strings.ToLower(p), "proto=") {
				return strings.Trim(strings.SplitN(p, "=", 2)[1], `"`)
			}
		}
	}
	// 3) Conexión TLS local
	if r.TLS != nil {
		return "https"
	}
	return "http"
}
