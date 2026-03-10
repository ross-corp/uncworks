package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/uncworks/aot/internal/sidecar"
)

func main() {
	port := 50052
	if p := os.Getenv("AOT_SIDECAR_PORT"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil {
			port = parsed
		}
	}

	gw := sidecar.NewGateway(port)

	go func() {
		if err := gw.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Gateway failed: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down RPC Gateway...")
	gw.Stop()
}
