package nexus

import (
	"net/http"
)

func ApplyMiddlewares(mux http.Handler, middlewares []func(next http.Handler) http.Handler) http.Handler {
	if Server.Secret != "" {
		mux = ValidateSecret(mux)
	}
	for i := len(middlewares) - 1; i >= 0; i-- {
		mux = middlewares[i](mux)
	}
	return mux
}

func ValidateSecret(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ok := Server.EndpointIsPublic(r)

		if ok {
			next.ServeHTTP(w, r)
			return
		}

		/**
		Evaluar secret
		*/
		secret := r.Header.Get("x-secret")

		if secret == "" || secret != Server.Secret {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
