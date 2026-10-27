package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"testgo/internal/models"
	"testgo/internal/repository"

	"github.com/go-chi/chi/v5"
)

var allowedSorts = map[string]struct{}{
	"name":   {},
	"-name":  {},
	"price":  {},
	"-price": {},
	"stock":  {},
	"-stock": {},
}

var allowedListParams = map[string]struct{}{
	"after":     {},
	"limit":     {},
	"sort":      {},
	"name":      {},
	"stock_min": {},
	"price_min": {},
	"price_max": {},
}

// ProductHandler menangani endpoint CRUD produk.
type ProductHandler struct {
	repo repository.ProductRepository
}

func NewProductHandler(repo repository.ProductRepository) *ProductHandler {
	return &ProductHandler{repo: repo}
}

func (h *ProductHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/{uuid}", h.Get)
	r.Put("/{uuid}", h.Update)
	r.Patch("/{uuid}", h.Update)
	r.Delete("/{uuid}", h.Delete)

	return r
}

// List godoc: GET /products?after=<uuid>&limit=20&sort=-price&name=kopi
func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) {
	params, paramErr := parseListParams(r)
	if paramErr != nil {
		writeError(w, paramErr.status, paramErr.message)
		return
	}

	page, err := h.repo.List(r.Context(), params)
	if err != nil {
		h.handleRepoError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":        page.Items,
		"next_cursor": page.NextCursor,
		"has_more":    page.HasMore,
	})
}

// Get godoc: GET /products/{uuid}
func (h *ProductHandler) Get(w http.ResponseWriter, r *http.Request) {
	product, err := h.repo.GetByUUID(r.Context(), chi.URLParam(r, "uuid"))
	if err != nil {
		h.handleRepoError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": product})
}

// Create godoc: POST /products
func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateProductRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		writeValidationError(w, err)
		return
	}

	product, err := h.repo.Create(r.Context(), req)
	if err != nil {
		h.handleRepoError(w, err)
		return
	}

	w.Header().Set("Location", "/products/"+product.UUID)
	writeJSON(w, http.StatusCreated, map[string]any{"data": product})
}

// Update godoc: PUT/PATCH /products/{uuid}
func (h *ProductHandler) Update(w http.ResponseWriter, r *http.Request) {
	var req models.UpdateProductRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		writeValidationError(w, err)
		return
	}

	product, err := h.repo.Update(r.Context(), chi.URLParam(r, "uuid"), req)
	if err != nil {
		h.handleRepoError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": product})
}

// Delete godoc: DELETE /products/{uuid}
func (h *ProductHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.repo.Delete(r.Context(), chi.URLParam(r, "uuid")); err != nil {
		h.handleRepoError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type paramError struct {
	status  int
	message string
}

func parseListParams(r *http.Request) (repository.ListParams, *paramError) {
	query := r.URL.Query()

	for key := range query {
		if _, ok := allowedListParams[key]; !ok {
			return repository.ListParams{}, &paramError{
				status:  http.StatusBadRequest,
				message: "parameter tidak dikenal: " + key,
			}
		}
	}

	params := repository.ListParams{
		After: query.Get("after"),
		Limit: repository.DefaultLimit,
		Sort:  query.Get("sort"),
		Name:  strings.TrimSpace(query.Get("name")),
	}

	if raw := query.Get("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > repository.MaxLimit {
			return repository.ListParams{}, &paramError{
				status:  http.StatusUnprocessableEntity,
				message: "limit harus antara 1 dan 100",
			}
		}
		params.Limit = limit
	}

	if params.Sort != "" {
		if _, ok := allowedSorts[params.Sort]; !ok {
			return repository.ListParams{}, &paramError{
				status:  http.StatusBadRequest,
				message: "sort tidak valid",
			}
		}
	}

	if raw := query.Get("stock_min"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			return repository.ListParams{}, &paramError{
				status:  http.StatusBadRequest,
				message: "stock_min tidak valid",
			}
		}
		params.StockMin = &value
	}

	if raw := query.Get("price_min"); raw != "" {
		value, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return repository.ListParams{}, &paramError{
				status:  http.StatusBadRequest,
				message: "price_min tidak valid",
			}
		}
		params.PriceMin = &value
	}

	if raw := query.Get("price_max"); raw != "" {
		value, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return repository.ListParams{}, &paramError{
				status:  http.StatusBadRequest,
				message: "price_max tidak valid",
			}
		}
		params.PriceMax = &value
	}

	return params, nil
}

func (h *ProductHandler) handleRepoError(w http.ResponseWriter, err error) {
	if errors.Is(err, repository.ErrInvalidCursor) {
		writeError(w, http.StatusBadRequest, "cursor tidak valid")
		return
	}

	if errors.Is(err, repository.ErrNotFound) {
		writeError(w, http.StatusNotFound, "produk tidak ditemukan")
		return
	}

	slog.Error("repository error", "error", err)
	writeError(w, http.StatusInternalServerError, "terjadi kesalahan pada server")
}

func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return errors.New("body JSON tidak valid: " + err.Error())
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("gagal encode response", "error", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{"message": message},
	})
}

func writeValidationError(w http.ResponseWriter, err error) {
	var verrs models.ValidationErrors
	if errors.As(err, &verrs) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"error": map[string]any{
				"message": "validasi gagal",
				"fields":  verrs,
			},
		})
		return
	}

	writeError(w, http.StatusUnprocessableEntity, err.Error())
}
