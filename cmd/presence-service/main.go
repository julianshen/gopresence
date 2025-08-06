package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"

	"gopresence/internal/auth"
	"gopresence/internal/config"
	"gopresence/internal/handlers"
	"gopresence/internal/metrics"
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
	// Metrics endpoint
	r.Handle("/metrics", metrics.Handler())

	// Health routes
	hh := handlers.NewHealthHandler(svc)
	r.HandleFunc("/health/liveness", hh.Liveness).Methods(http.MethodGet)
	r.HandleFunc("/health/readiness", hh.Readiness).Methods(http.MethodGet)

	// API routes (instrumented)
	ph := handlers.NewPresenceHandler(svc)
	r.Handle("/api/v2/presence/{user_id}", metrics.Middleware("presence.user", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
		switch r.Method {
		case http.MethodGet:
			ph.GetPresence(w, r)
		case http.MethodPut:
			ph.SetPresence(w, r)
		case http.MethodOptions:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}), svc.Cache())).Methods(http.MethodGet, http.MethodPut, http.MethodOptions)
	r.Handle("/api/v2/presence", metrics.Middleware("presence.multi", http.HandlerFunc(ph.GetMultiplePresences), svc.Cache())).Methods(http.MethodGet, http.MethodOptions)
	r.Handle("/api/v2/presence/batch", metrics.Middleware("presence.batch", http.HandlerFunc(ph.BatchPresence), svc.Cache())).Methods(http.MethodPost, http.MethodOptions)

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