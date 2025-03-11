package nexus

import (
	"fmt"
	"net/http"
)

func (server *ServerStruct) ApplyMiddlewares(mux http.Handler) http.Handler {
	fmt.Println("ApplyMiddlewares", server.ServerName, server.Debug)
	if server.Debug {
		mux = server.LogRequest(mux)
	}

	if server.Secret != "" {
		mux = server.ValidateSecret(mux)
	}
	for i := len(server.Middlewares) - 1; i >= 0; i-- {
		mux = server.Middlewares[i](mux, server)
	}
	return mux
}

func (server *ServerStruct) LogRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("[%s] %s %s\n", server.ServerName, r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func (server *ServerStruct) ValidateSecret(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		ok := server.EndpointIsPublic(r)

		if ok {
			next.ServeHTTP(w, r)
			return
		}

		/**
		Evaluar secret
		*/
		secret := r.Header.Get("x-secret")

		if secret == "" || secret != server.Secret {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
