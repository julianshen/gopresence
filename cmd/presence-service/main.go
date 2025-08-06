package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"

	"gopresence/internal/auth"
	"gopresence/internal/config"
	"gopresence/internal/handlers"
	"gopresence/internal/service"
)

func main(){
	cfg, err := config.Load()
	if err != nil { log.Fatalf("config load: %v", err) }

	// Build service
	builder := service.NewServiceBuilder(cfg)
	svc, err := builder.Build()
	if err != nil { log.Fatalf("service build: %v", err) }
	defer svc.Close()

	// Router
	r := mux.NewRouter()
	// API routes
	ph := handlers.NewPresenceHandler(svc)
	r.HandleFunc("/api/v2/presence/{user_id}", ph.GetPresence).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/api/v2/presence/{user_id}", ph.SetPresence).Methods(http.MethodPut, http.MethodOptions)
	r.HandleFunc("/api/v2/presence", ph.GetMultiplePresences).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/api/v2/presence/batch", ph.BatchPresence).Methods(http.MethodPost, http.MethodOptions)

	// Middlewares: CORS -> Auth (example uses optional auth for demonstration)
	var handler http.Handler = r
	handler = handlers.CORSMiddleware(handler)
	jwtmw := auth.NewJWTMiddleware(cfg.Auth.JWTSecret, cfg.Auth.JWTIssuer)
	handler = jwtmw.OptionalAuthenticate(handler)

	port := os.Getenv("SERVICE_PORT")
	if port == "" { port = "8080" }
	log.Printf("starting presence-service on :%s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("listen: %v", err)
	}
}