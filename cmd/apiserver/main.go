package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"connectrpc.com/connect"
	"connectrpc.com/grpchealth"
	"connectrpc.com/grpcreflect"
	"connectrpc.com/validate"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/uncworks/aot/gen/go/api/v1/apiv1connect"
	"github.com/uncworks/aot/internal/server"
)

func main() {
	addr := ":50051"

	svc := server.NewAOTServiceHandler()

	// Protovalidate interceptor rejects invalid requests with INVALID_ARGUMENT
	validateInterceptor := validate.NewInterceptor()

	mux := http.NewServeMux()

	// Register AOTService handler
	path, handler := apiv1connect.NewAOTServiceHandler(svc,
		connect.WithInterceptors(validateInterceptor),
	)
	mux.Handle(path, handler)

	// Health check
	checker := grpchealth.NewStaticChecker(apiv1connect.AOTServiceName)
	mux.Handle(grpchealth.NewHandler(checker))

	// Reflection for grpcurl compatibility
	reflector := grpcreflect.NewStaticReflector(apiv1connect.AOTServiceName)
	mux.Handle(grpcreflect.NewHandlerV1(reflector))
	mux.Handle(grpcreflect.NewHandlerV1Alpha(reflector))

	httpServer := &http.Server{
		Addr: addr,
		// h2c enables HTTP/2 without TLS for gRPC clients
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}

	go func() {
		log.Printf("AOT API server listening on %s (gRPC + Connect + gRPC-Web)", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down...")
	if err := httpServer.Shutdown(context.Background()); err != nil {
		log.Printf("Shutdown error: %v", err)
	}
}
