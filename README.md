# nexus

```go
package main

import (
	"fmt"
	"github.com/codecraftkit/flash-server/internal/config"
	"github.com/codecraftkit/flash-server/internal/handlers"
	"github.com/codecraftkit/nexus"
	"log"
	"os"
)

func main() {
	config.LoadEnv()
	config.MsCfg.Secret = os.Getenv("SECRET")

	serverSetting := nexus.ServerStruct{
		Secret: os.Getenv("SECRET"),
		Port:   os.Getenv("PORT"),
		Debug:  true,
		Endpoints: [][]nexus.EndpointPath{
			handlers.HomeEndpoints,
		},
	}

	httpServer := nexus.Server.Create(serverSetting)

	fmt.Printf("Server running on port %s\n", httpServer.Addr)
	if err := httpServer.ListenAndServe(); err != nil {
		log.Fatal(err)
	}

}

```