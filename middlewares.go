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
	//if server.Secret != "" && server.SecretMiddleware == nil && !server.Settings.IgnoreSecret {
	//	mux = server.ValidateSecret(mux)
	//}

	//fmt.Println(server.Settings.IgnoreSecret)
	//if server.SecretMiddleware != nil {
	//	mux = server.SecretMiddleware(mux, server)
	//}

	// If the server is in debug mode, the server will be register the LogRequest middleware that will log the request on the console
	if server.Debug {
		mux = server.LogRequest(mux)
	}

	return mux
}

// LogRequest log the request on the console
func (server *Server) LogRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if server.Debug && r.URL.Path != "/_health" {
			fmt.Printf("[%s] %s %s\n", server.ServerName, r.Method, r.URL.Path)
		}
		_, ok := server.GetEndpoint(r)
		if !ok {
			http.Error(w, "Endpoint not found", http.StatusNotFound)
			return
		}
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

//func (server *Server) CorsMiddleware(next http.Handler) http.Handler {
//	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		origin := r.Header.Get("Origin")
//
//		fmt.Println("[LOG]Origin", origin, strings.Contains(origin, "http://localhost"))
//
//		// Permitir tanto localhost como IP local
//		if isOriginAllowed(origin) {
//			w.Header().Set("Access-Control-Allow-Origin", origin)
//			w.Header().Set("Access-Control-Allow-Credentials", "true")
//			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
//			w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, X-Session, X-Secret, X-Token, X-Organization-App-Integration-Id")
//			//w.Header().Set("Access-Control-Allow-Headers", "*")
//		}
//
//		if r.Method == http.MethodOptions {
//			w.WriteHeader(http.StatusOK)
//			return
//		}
//
//		next.ServeHTTP(w, r)
//	})
//}
//
//func isOriginAllowed(origin string) bool {
//	allowed := []string{
//		"http://localhost:8000",
//		"http://10.2.20.200:8000",
//		"http://10.2.20.104:8000",
//		// puedes agregar más
//	}
//
//	for _, o := range allowed {
//		if o == origin {
//			return true
//		}
//	}
//	return false
//}
