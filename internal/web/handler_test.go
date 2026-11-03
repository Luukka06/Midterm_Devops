package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"stockpilot/internal/inventory"
)

func fixture(t *testing.T) http.Handler {
	t.Helper()
	store, err := inventory.NewStore(filepath.Join(t.TempDir(), "items.json"))
	if err != nil {
		t.Fatal(err)
	}
	return NewHandler(store)
}

func send(t *testing.T, handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	result := httptest.NewRecorder()
	handler.ServeHTTP(result, request)
	return result
}

func TestHealthCheck(t *testing.T) {
	response := send(t, fixture(t), "GET", "/healthz", "")
	if response.Code != 200 || !strings.Contains(response.Body.String(), `"status":"ok"`) {
		t.Fatalf("unexpected health response: %d %s", response.Code, response.Body.String())
	}
}

func TestHomePage(t *testing.T) {
	response := send(t, fixture(t), "GET", "/", "")
	if response.Code != 200 || !strings.Contains(response.Body.String(), "StockPilot") {
		t.Fatalf("webpage unavailable: %d", response.Code)
	}
}

func TestStaticJavaScript(t *testing.T) {
	response := send(t, fixture(t), "GET", "/app.js", "")
	if response.Code != 200 || !strings.Contains(response.Body.String(), "renderItems") {
		t.Fatalf("javascript missing: %d", response.Code)
	}
}

func TestListItems(t *testing.T) {
	response := send(t, fixture(t), "GET", "/api/items", "")
	var items []inventory.Item
	if err := json.Unmarshal(response.Body.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	if response.Code != 200 || len(items) != 7 {
		t.Fatalf("status %d, items %d", response.Code, len(items))
	}
}

func TestSummaryEndpoint(t *testing.T) {
	response := send(t, fixture(t), "GET", "/api/summary", "")
	var summary inventory.Summary
	if err := json.Unmarshal(response.Body.Bytes(), &summary); err != nil {
		t.Fatal(err)
	}
	if summary.TotalItems != 7 || summary.OutOfStock != 1 || summary.LowStock != 1 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
}

func TestCreateItem(t *testing.T) {
	response := send(t, fixture(t), "POST", "/api/items", `{"name":"Tape","quantity":12,"min_stock":5}`)
	if response.Code != 201 {
		t.Fatalf("status %d: %s", response.Code, response.Body)
	}
	if !strings.Contains(response.Body.String(), "Tape") {
		t.Fatal("missing created item")
	}
}

func TestCreateItemRejectsNegativeQuantity(t *testing.T) {
	response := send(t, fixture(t), "POST", "/api/items", `{"name":"Tape","quantity":-3,"min_stock":5}`)
	if response.Code != 400 {
		t.Fatalf("unexpected status: %d", response.Code)
	}
}

func TestCreateRejectsUnknownField(t *testing.T) {
	response := send(t, fixture(t), "POST", "/api/items", `{"name":"Tape","fake":true}`)
	if response.Code != 400 {
		t.Fatalf("unexpected status: %d", response.Code)
	}
}

func TestCreateRejectsWrongContentType(t *testing.T) {
	response := send(t, fixture(t), "POST", "/api/items", "")
	if response.Code != 415 {
		t.Fatalf("unexpected status: %d", response.Code)
	}
}

func TestUpdateAndDeleteItem(t *testing.T) {
	handler := fixture(t)
	updated := send(t, handler, "PATCH", "/api/items/SP-101", `{"quantity":3}`)
	if updated.Code != 200 || !strings.Contains(updated.Body.String(), `"quantity":3`) {
		t.Fatalf("update failed: %d %s", updated.Code, updated.Body.String())
	}
	removed := send(t, handler, "DELETE", "/api/items/SP-101", "")
	if removed.Code != 204 {
		t.Fatalf("delete status %d", removed.Code)
	}
	missing := send(t, handler, "DELETE", "/api/items/SP-101", "")
	if missing.Code != 404 {
		t.Fatalf("missing item status %d", missing.Code)
	}
}

func TestUpdateRequiresField(t *testing.T) {
	response := send(t, fixture(t), "PATCH", "/api/items/SP-101", `{}`)
	if response.Code != 400 {
		t.Fatalf("unexpected status: %d", response.Code)
	}
}

func TestCreateRejectsTwoJSONObjects(t *testing.T) {
	response := send(t, fixture(t), "POST", "/api/items", `{"name":"Tape"}{"name":"Pencil"}`)
	if response.Code != 400 {
		t.Fatalf("unexpected status: %d", response.Code)
	}
}
