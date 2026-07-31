package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/yourname/expense-tracker/src/handlers"
	"github.com/yourname/expense-tracker/src/service"
	"github.com/yourname/expense-tracker/src/store"
)

func main() {
	dataFile := os.Getenv("DATA_FILE")
	if dataFile == "" {
		dataFile = filepath.Join("data", "expenses.json")
	}
	if err := os.MkdirAll(filepath.Dir(dataFile), 0755); err != nil {
		log.Fatalf("create data directory: %v", err)
	}
	persister := store.NewPersister(dataFile)
	expenseStore := store.NewExpenseStore(persister)
	if err := expenseStore.LoadFromDisk(); err != nil {
		log.Fatalf("load data: %v", err)
	}

	svc := service.NewExpenseService(expenseStore)
	h := handlers.NewExpenseHandler(svc)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /expenses", h.Create)
	mux.HandleFunc("GET /expenses", h.List)
	mux.HandleFunc("GET /expenses/total", h.Total)
	mux.HandleFunc("DELETE /expenses/{id}", h.Delete)
	mux.HandleFunc("DELETE /expenses/", handlers.MissingID)
	mux.HandleFunc("/expenses", handlers.MethodNotAllowed)
	mux.HandleFunc("/expenses/", handlers.MethodNotAllowed)
	mux.HandleFunc("/", handlers.NotFound)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	server := &http.Server{Addr: "0.0.0.0:" + port, Handler: recoverMiddleware(mux)}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() { <-ctx.Done(); shutdown(server, persister, expenseStore) }()
	log.Printf("server listening on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func shutdown(server *http.Server, persister *store.Persister, expenseStore *store.ExpenseStore) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("shutdown server: %v", err)
	}
	if err := persister.Save(expenseStore.All()); err != nil {
		log.Printf("final save: %v", err)
	}
	log.Println("server shut down cleanly")
}

func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("panic: %v", recovered)
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal server error"})
			}
		}()
		next.ServeHTTP(w, r)
	})
}
