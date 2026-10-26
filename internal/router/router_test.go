package router_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"testgo/internal/repository"
	"testgo/internal/router"
)

func newTestServer(t *testing.T) http.Handler {
	t.Helper()
	return router.New(repository.NewMemoryProductRepository())
}

func doRequest(t *testing.T, h http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}

	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestProductCRUDFlow(t *testing.T) {
	h := newTestServer(t)

	// Create
	rec := doRequest(t, h, http.MethodPost, "/products", map[string]any{
		"name":  "Kopi Gayo",
		"price": 45000,
		"stock": 10,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var created struct {
		Data struct {
			ID    string  `json:"id"`
			Name  string  `json:"name"`
			Price float64 `json:"price"`
			Stock int     `json:"stock"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	if created.Data.ID == "" || created.Data.Name != "Kopi Gayo" {
		t.Fatalf("unexpected created product: %+v", created.Data)
	}

	id := created.Data.ID

	// List
	rec = doRequest(t, h, http.MethodGet, "/products", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list: status = %d, body = %s", rec.Code, rec.Body.String())
	}

	// Get
	rec = doRequest(t, h, http.MethodGet, "/products/"+id, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("get: status = %d, body = %s", rec.Code, rec.Body.String())
	}

	// Update (partial)
	rec = doRequest(t, h, http.MethodPatch, "/products/"+id, map[string]any{"stock": 5})
	if rec.Code != http.StatusOK {
		t.Fatalf("update: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var updated struct {
		Data struct {
			Stock int `json:"stock"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&updated); err != nil {
		t.Fatalf("decode update: %v", err)
	}
	if updated.Data.Stock != 5 {
		t.Fatalf("stock = %d, want 5", updated.Data.Stock)
	}

	// Delete
	rec = doRequest(t, h, http.MethodDelete, "/products/"+id, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete: status = %d, body = %s", rec.Code, rec.Body.String())
	}

	// Get after delete -> 404
	rec = doRequest(t, h, http.MethodGet, "/products/"+id, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("get after delete: status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestCreateProductValidation(t *testing.T) {
	h := newTestServer(t)

	rec := doRequest(t, h, http.MethodPost, "/products", map[string]any{
		"name":  "",
		"price": -1,
	})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusUnprocessableEntity, rec.Body.String())
	}
}

func TestInvalidJSON(t *testing.T) {
	h := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewBufferString("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}
