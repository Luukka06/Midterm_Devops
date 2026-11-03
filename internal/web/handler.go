package web

import (
	"embed"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"strings"

	"stockpilot/internal/inventory"
)

//go:embed static/*
var assets embed.FS

type responseError struct {
	Error string `json:"error"`
}

func NewHandler(store *inventory.Store) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		sendJSON(w, 200, map[string]string{"status": "ok", "service": "stockpilot"})
	})
	mux.HandleFunc("GET /api/items", func(w http.ResponseWriter, r *http.Request) {
		items := store.List()
		if items == nil {
			items = []inventory.Item{}
		}
		sendJSON(w, 200, items)
	})
	mux.HandleFunc("GET /api/summary", func(w http.ResponseWriter, r *http.Request) {
		sendJSON(w, 200, store.Summary())
	})
	mux.HandleFunc("POST /api/items", func(w http.ResponseWriter, r *http.Request) {
		var in inventory.ItemInput
		if !decodeJSON(w, r, &in) {
			return
		}
		item, err := store.Add(in)
		if err != nil {
			sendError(w, err)
			return
		}
		sendJSON(w, 201, item)
	})
	mux.HandleFunc("PATCH /api/items/{id}", func(w http.ResponseWriter, r *http.Request) {
		var in inventory.UpdateInput
		if !decodeJSON(w, r, &in) {
			return
		}
		item, err := store.Update(r.PathValue("id"), in)
		if err != nil {
			sendError(w, err)
			return
		}
		sendJSON(w, 200, item)
	})
	mux.HandleFunc("DELETE /api/items/{id}", func(w http.ResponseWriter, r *http.Request) {
		if err := store.Delete(r.PathValue("id")); err != nil {
			sendError(w, err)
			return
		}
		w.WriteHeader(204)
	})

	static, _ := fs.Sub(assets, "static")
	mux.Handle("/", http.FileServer(http.FS(static)))
	return mux
}

func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		sendJSON(w, 415, responseError{Error: "Content-Type must be application/json"})
		return false
	}
	decoder := json.NewDecoder(io.LimitReader(r.Body, 64*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(v); err != nil {
		sendJSON(w, 400, responseError{Error: "Invalid JSON request"})
		return false
	}
	if decoder.Decode(new(any)) != io.EOF {
		sendJSON(w, 400, responseError{Error: "Only one JSON value is allowed"})
		return false
	}
	return true
}

func sendError(w http.ResponseWriter, err error) {
	if errors.Is(err, inventory.ErrInvalid) {
		sendJSON(w, 400, responseError{Error: "Check the item values and try again"})
		return
	}
	if errors.Is(err, inventory.ErrNotFound) {
		sendJSON(w, 404, responseError{Error: "Item not found"})
		return
	}
	sendJSON(w, 500, responseError{Error: "Unable to save item"})
}

func sendJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
