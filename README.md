# nexus

**Nexus** is a Go library designed to simplify the creation of HTTP servers by leveraging Go's standard library. It provides developers with a streamlined interface to set up robust and efficient web servers without relying on external frameworks. By utilizing Go's native capabilities, Nexus ensures optimal performance and seamless integration within the Go ecosystem.

## Install

```bash
go get github.com/codecraftkit/nexus
```

```go
package main

import (
	"github.com/codecraftkit/flash-server/internal/config"
	"github.com/codecraftkit/nexus"
	"net/http"
	"os"
)

func main() {
	config.LoadEnv()

	server := &nexus.ServerStruct{
		//Secret:      os.Getenv("SECRET"),
		Port:        os.Getenv("PORT"),
		Debug:       true,
		Middlewares: []func(next http.Handler) http.Handler{
			//VerifySession,
		},
		Endpoints: [][]nexus.EndpointPath{
			HomeEndpoints,
			UserEndpoints,
		},
	}

	nexus.Server.Create(server)

}

func Home(w http.ResponseWriter, r *http.Request) {

	w.Write([]byte("Home"))

}

func Users(w http.ResponseWriter, r *http.Request) {

	users := []struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}{
		{Name: "John", Age: 30},
		{Name: "Mary", Age: 25},
		{Name: "Peter", Age: 40},
	}

	nexus.ResponseWithJSON(w, http.StatusOK, users)

}

func UsersSave(w http.ResponseWriter, r *http.Request) {

	// save user

	// ...
	nexus.ResponseWithJSON(w, http.StatusOK, map[string]string{"message": "User saved"})

}

var HomeEndpoints = []nexus.EndpointPath{
	{Path: "GET /home", HandlerFunc: Home},
}

var UserEndpoints = []nexus.EndpointPath{
	{Path: "GET /users", HandlerFunc: Users},
	{Path: "POST /users", HandlerFunc: UsersSave},
}

func VerifySession(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session := r.Header.Get("x-session")
		if session == "" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})

}

```