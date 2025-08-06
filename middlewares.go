package nexus

import (
	"fmt"
	"net/http"
)

// ApplyMiddlewares apply all middlewares to the mux;
// if the server is in debug mode, the server will be register the LogRequest middleware that will log the request on the console
// if the server has a secret, the server will be register the ValidateSecret middleware that will check if the request has a secret
func (server *Server) ApplyMiddlewares(mux http.Handler) http.Handler {

	// Apply all middlewares
	for i := len(server.Middlewares) - 1; i >= 0; i-- {
		mux = server.Middlewares[i](mux, server)
	}

	// If the server has a secret, the server will be register the ValidateSecret middleware that will check if the request has a secret
	if server.Secret != "" && server.SecretMiddleware == nil {
		mux = server.ValidateSecret(mux)
	}

	if server.SecretMiddleware != nil {
		mux = server.SecretMiddleware(mux, server)
	}

	// If the server is in debug mode, the server will be register the LogRequest middleware that will log the request on the console
	if server.Debug {
		mux = server.LogRequest(mux)
	}

	return mux
}

// LogRequest log the request on the console
func (server *Server) LogRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("[%s] %s %s\n", server.ServerName, r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

// ValidateSecret check if the request has a secret
func (server *Server) ValidateSecret(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		ok := server.EndpointIsPublic(r)

		if ok {
			next.ServeHTTP(w, r)
			return
		}

		// Evaluate secret
		secret := r.Header.Get("x-secret")

		// If the secret is empty or not equal to the server's secret, the request is unauthorized (Status 401)
		// TODO: optimize this check
		if secret == "" || secret != server.Secret {
			http.Error(w, "Unauthorized: Invalid secret [nexus]", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
