package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/stygian-tech/anypub-kit/xrpc"
)

func main() {
	mux := http.NewServeMux()
	mux.Handle("/xrpc/"+xrpc.NSID, xrpc.Handler())
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8080"
	}
	server := &http.Server{Addr: ":" + port, Handler: mux, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 32 << 10}
	log.Printf("AnyPubKit preparation service listening on %s", server.Addr)
	log.Fatal(server.ListenAndServe())
}
