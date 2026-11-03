package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "path/filepath"
    "syscall"
    "time"

    "stockpilot/internal/inventory"
    "stockpilot/internal/web"
)

func main() {
    address := env("STOCKPILOT_ADDR", "127.0.0.1:8080")
    dataFile := env("STOCKPILOT_DATA", filepath.Join("data", "items.json"))

    store, err := inventory.NewStore(dataFile)
    if err != nil {
        log.Fatalf("open inventory: %v", err)
    }

    server := &http.Server{
        Addr:              address,
        Handler:           web.NewHandler(store),
        ReadHeaderTimeout: 5 * time.Second,
    }

    done := make(chan os.Signal, 1)
    signal.Notify(done, os.Interrupt, syscall.SIGTERM)

    go func() {
        log.Printf("StockPilot listening at http://%s", address)
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("web server: %v", err)
        }
    }()

    <-done
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    if err := server.Shutdown(ctx); err != nil {
        log.Printf("shutdown: %v", err)
    }
}

func env(key, fallback string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return fallback
}
