package router_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"testgo/internal/repository"
	"testgo/internal/router"

	"github.com/google/uuid"
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

type productData struct {
	UUID  string  `json:"uuid"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
	Stock int     `json:"stock"`
}

type productResponse struct {
	Data productData `json:"data"`
}

type listResponse struct {
	Data       []productData `json:"data"`
	NextCursor *string       `json:"next_cursor"`
	HasMore    bool          `json:"has_more"`
}

func decodeInto[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()

	var out T
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v, body = %s", err, rec.Body.String())
	}
	return out
}

func createProduct(t *testing.T, h http.Handler, name string, price float64, stock int) string {
	t.Helper()

	rec := doRequest(t, h, http.MethodPost, "/products", map[string]any{
		"name":  name,
		"price": price,
		"stock": stock,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create %q: status = %d, body = %s", name, rec.Code, rec.Body.String())
	}

	resp := decodeInto[productResponse](t, rec)
	if resp.Data.UUID == "" {
		t.Fatalf("create %q: uuid kosong", name)
	}
	return resp.Data.UUID
}

func listProducts(t *testing.T, h http.Handler, path string) listResponse {
	t.Helper()

	rec := doRequest(t, h, http.MethodGet, path, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list %q: status = %d, body = %s", path, rec.Code, rec.Body.String())
	}
	return decodeInto[listResponse](t, rec)
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

	created := decodeInto[productResponse](t, rec)
	if created.Data.UUID == "" || created.Data.Name != "Kopi Gayo" {
		t.Fatalf("unexpected created product: %+v", created.Data)
	}
	if location := rec.Header().Get("Location"); location != "/products/"+created.Data.UUID {
		t.Fatalf("location = %q, want %q", location, "/products/"+created.Data.UUID)
	}

	productUUID := created.Data.UUID

	// List halaman pertama
	list := listProducts(t, h, "/products")
	if len(list.Data) != 1 || list.Data[0].UUID != productUUID {
		t.Fatalf("unexpected list: %+v", list)
	}
	if list.HasMore {
		t.Fatalf("has_more = true, want false")
	}
	if list.NextCursor != nil {
		t.Fatalf("next_cursor = %v, want null", *list.NextCursor)
	}

	// Get
	rec = doRequest(t, h, http.MethodGet, "/products/"+productUUID, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("get: status = %d, body = %s", rec.Code, rec.Body.String())
	}

	// Update (partial)
	rec = doRequest(t, h, http.MethodPatch, "/products/"+productUUID, map[string]any{"stock": 5})
	if rec.Code != http.StatusOK {
		t.Fatalf("update: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	updated := decodeInto[productResponse](t, rec)
	if updated.Data.Stock != 5 {
		t.Fatalf("stock = %d, want 5", updated.Data.Stock)
	}

	// Delete (soft delete)
	rec = doRequest(t, h, http.MethodDelete, "/products/"+productUUID, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete: status = %d, body = %s", rec.Code, rec.Body.String())
	}

	// Get after delete -> 404
	rec = doRequest(t, h, http.MethodGet, "/products/"+productUUID, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("get after delete: status = %d, body = %s", rec.Code, rec.Body.String())
	}

	// List after delete -> kosong
	list = listProducts(t, h, "/products")
	if len(list.Data) != 0 {
		t.Fatalf("list after delete = %+v, want kosong", list.Data)
	}
}

func TestListPagination(t *testing.T) {
	h := newTestServer(t)

	first := createProduct(t, h, "Produk Pertama", 10000, 1)
	second := createProduct(t, h, "Produk Kedua", 20000, 2)
	third := createProduct(t, h, "Produk Ketiga", 30000, 3)

	// Default sort id DESC: produk terbaru lebih dulu.
	page1 := listProducts(t, h, "/products?limit=2")
	if len(page1.Data) != 2 {
		t.Fatalf("page 1: len = %d, want 2", len(page1.Data))
	}
	if !page1.HasMore {
		t.Fatalf("page 1: has_more = false, want true")
	}
	if page1.NextCursor == nil || *page1.NextCursor != second {
		t.Fatalf("page 1: next_cursor = %v, want %q", page1.NextCursor, second)
	}
	if page1.Data[0].UUID != third || page1.Data[1].UUID != second {
		t.Fatalf("page 1: urutan tidak sesuai: %+v", page1.Data)
	}

	page2 := listProducts(t, h, "/products?limit=2&after="+*page1.NextCursor)
	if len(page2.Data) != 1 || page2.Data[0].UUID != first {
		t.Fatalf("page 2: %+v, want hanya %q", page2.Data, first)
	}
	if page2.HasMore {
		t.Fatalf("page 2: has_more = true, want false")
	}
	if page2.NextCursor != nil {
		t.Fatalf("page 2: next_cursor = %v, want null", *page2.NextCursor)
	}
}

func TestListSortAndFilter(t *testing.T) {
	h := newTestServer(t)

	gayo := createProduct(t, h, "Kopi Gayo", 45000, 10)
	teh := createProduct(t, h, "Teh Melati", 20000, 5)
	robusta := createProduct(t, h, "Kopi Robusta", 30000, 0)
	susu := createProduct(t, h, "Susu", 15000, 20)

	assertUUIDs := func(t *testing.T, got []productData, want ...string) {
		t.Helper()
		if len(got) != len(want) {
			t.Fatalf("len = %d, want %d (%+v)", len(got), len(want), got)
		}
		for i, id := range want {
			if got[i].UUID != id {
				t.Fatalf("item[%d] = %q, want %q", i, got[i].UUID, id)
			}
		}
	}

	// Filter name (contains, case-insensitive); default sort id DESC
	assertUUIDs(t, listProducts(t, h, "/products?name=kopi").Data, robusta, gayo)

	// Filter stock_min
	assertUUIDs(t, listProducts(t, h, "/products?stock_min=10").Data, susu, gayo)

	// Filter price_min + price_max (inklusif)
	assertUUIDs(t, listProducts(t, h, "/products?price_min=20000&price_max=30000").Data, robusta, teh)

	// Sort price ASC
	assertUUIDs(t, listProducts(t, h, "/products?sort=price").Data, susu, teh, robusta, gayo)

	// Sort name DESC
	assertUUIDs(t, listProducts(t, h, "/products?sort=-name").Data, teh, susu, robusta, gayo)

	// Kombinasi filter + sort + cursor
	page1 := listProducts(t, h, "/products?sort=price&limit=2")
	assertUUIDs(t, page1.Data, susu, teh)
	if !page1.HasMore || page1.NextCursor == nil {
		t.Fatalf("page 1: has_more = %v, next_cursor = %v", page1.HasMore, page1.NextCursor)
	}

	page2 := listProducts(t, h, "/products?sort=price&limit=2&after="+*page1.NextCursor)
	assertUUIDs(t, page2.Data, robusta, gayo)
	if page2.HasMore {
		t.Fatalf("page 2: has_more = true, want false")
	}
}

func TestListInvalidParams(t *testing.T) {
	h := newTestServer(t)
	validUUID := createProduct(t, h, "Kopi Gayo", 45000, 10)
	missingUUID := uuid.NewString()

	doRequest(t, h, http.MethodDelete, "/products/"+validUUID, nil)

	cases := []struct {
		name       string
		path       string
		wantStatus int
		wantMsg    string
	}{
		{"after bukan uuid", "/products?after=bukan-uuid", http.StatusBadRequest, "cursor tidak valid"},
		{"after tidak ditemukan", "/products?after=" + missingUUID, http.StatusBadRequest, "cursor tidak valid"},
		{"after sudah dihapus", "/products?after=" + validUUID, http.StatusBadRequest, "cursor tidak valid"},
		{"limit nol", "/products?limit=0", http.StatusUnprocessableEntity, "limit harus antara 1 dan 100"},
		{"limit terlalu besar", "/products?limit=101", http.StatusUnprocessableEntity, "limit harus antara 1 dan 100"},
		{"limit bukan angka", "/products?limit=abc", http.StatusUnprocessableEntity, "limit harus antara 1 dan 100"},
		{"sort tidak dikenal", "/products?sort=uuid", http.StatusBadRequest, "sort tidak valid"},
		{"offset tidak diizinkan", "/products?offset=10", http.StatusBadRequest, "parameter tidak dikenal: offset"},
		{"page tidak diizinkan", "/products?page=2", http.StatusBadRequest, "parameter tidak dikenal: page"},
		{"stock_min bukan angka", "/products?stock_min=abc", http.StatusBadRequest, "stock_min tidak valid"},
		{"price_min bukan angka", "/products?price_min=abc", http.StatusBadRequest, "price_min tidak valid"},
		{"price_max bukan angka", "/products?price_max=abc", http.StatusBadRequest, "price_max tidak valid"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := doRequest(t, h, http.MethodGet, tc.path, nil)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d, body = %s", rec.Code, tc.wantStatus, rec.Body.String())
			}

			var body struct {
				Error struct {
					Message string `json:"message"`
				} `json:"error"`
			}
			if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body.Error.Message != tc.wantMsg {
				t.Fatalf("message = %q, want %q", body.Error.Message, tc.wantMsg)
			}
		})
	}
}

func TestGetInvalidUUID(t *testing.T) {
	h := newTestServer(t)

	rec := doRequest(t, h, http.MethodGet, "/products/bukan-uuid", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestCreateProductValidation(t *testing.T) {
	h := newTestServer(t)

	rec := doRequest(t, h, http.MethodPost, "/products", map[string]any{
		"name":  "",
		"price": -1,
		"stock": -1,
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

func TestResponseDoesNotLeakInternalFields(t *testing.T) {
	h := newTestServer(t)
	productUUID := createProduct(t, h, "Kopi Gayo", 45000, 10)

	requests := []struct {
		method string
		path   string
		body   any
	}{
		{http.MethodGet, "/products", nil},
		{http.MethodGet, "/products/" + productUUID, nil},
		{http.MethodGet, "/products?limit=1", nil},
		{http.MethodPatch, "/products/" + productUUID, map[string]any{"stock": 5}},
	}

	for _, req := range requests {
		rec := doRequest(t, h, req.method, req.path, req.body)
		if rec.Code < 200 || rec.Code >= 300 {
			t.Fatalf("%s %s: status = %d, body = %s", req.method, req.path, rec.Code, rec.Body.String())
		}

		var payload any
		if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
			t.Fatalf("%s %s: decode: %v", req.method, req.path, err)
		}
		assertNoInternalFields(t, payload, req.path)
	}
}

func assertNoInternalFields(t *testing.T, value any, path string) {
	t.Helper()

	switch v := value.(type) {
	case map[string]any:
		for key, child := range v {
			if key == "id" || strings.HasSuffix(key, "_id") ||
				strings.HasSuffix(key, "_by") || strings.HasSuffix(key, "_at") {
				t.Fatalf("field internal %q bocor di response %s", key, path)
			}
			assertNoInternalFields(t, child, path+"."+key)
		}
	case []any:
		for _, child := range v {
			assertNoInternalFields(t, child, path+"[]")
		}
	}
}
