package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"testgo/internal/models"
	"testgo/internal/repository"

	"github.com/go-chi/chi/v5"
)

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
	r.Get("/{id}", h.Get)
	r.Put("/{id}", h.Update)
	r.Patch("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)

	return r
}

// List godoc: GET /products
func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) {
	products, err := h.repo.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal mengambil data produk")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":  products,
		"total": len(products),
	})
}

// Get godoc: GET /products/{id}
func (h *ProductHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	product, err := h.repo.GetByID(r.Context(), id)
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

	w.Header().Set("Location", "/products/"+product.ID)
	writeJSON(w, http.StatusCreated, map[string]any{"data": product})
}

// Update godoc: PUT/PATCH /products/{id}
func (h *ProductHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req models.UpdateProductRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		writeValidationError(w, err)
		return
	}

	product, err := h.repo.Update(r.Context(), id, req)
	if err != nil {
		h.handleRepoError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": product})
}

// Delete godoc: DELETE /products/{id}
func (h *ProductHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.repo.Delete(r.Context(), id); err != nil {
		h.handleRepoError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *ProductHandler) handleRepoError(w http.ResponseWriter, err error) {
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
