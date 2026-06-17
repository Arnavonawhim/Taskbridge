package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"taskbridge/internal/api"
	"taskbridge/internal/store"
)

func main() {
	addr := flag.String("addr", ":8080", "server listen address")
	flag.Parse()
	memStore := store.NewMemoryStore()
	handler := api.NewHandler(memStore)
	mux := http.NewServeMux()
	api.RegisterRoutes(mux, handler)
	fmt.Printf("TaskBridge server listening on %s\n", *addr)
	log.Fatal(http.ListenAndServe(*addr, mux))
}
