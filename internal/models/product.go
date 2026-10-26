package models

import (
	"fmt"
	"strings"
	"time"
)

// Product adalah representasi produk yang disimpan dan dikembalikan via API.
type Product struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Price     float64   `json:"price"`
	Stock     int       `json:"stock"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateProductRequest adalah payload untuk membuat produk baru.
type CreateProductRequest struct {
	Name  string  `json:"name"`
	Price float64 `json:"price"`
	Stock int     `json:"stock"`
}

// UpdateProductRequest adalah payload untuk memperbarui produk.
// Field bernilai nil berarti tidak diubah (partial update).
type UpdateProductRequest struct {
	Name  *string  `json:"name"`
	Price *float64 `json:"price"`
	Stock *int     `json:"stock"`
}

// ValidationErrors memetakan nama field ke pesan kesalahannya.
type ValidationErrors map[string]string

func (v ValidationErrors) Error() string {
	parts := make([]string, 0, len(v))
	for field, msg := range v {
		parts = append(parts, field+": "+msg)
	}
	return strings.Join(parts, "; ")
}

func (r CreateProductRequest) Validate() error {
	errs := ValidationErrors{}

	if strings.TrimSpace(r.Name) == "" {
		errs["name"] = "wajib diisi"
	}
	if r.Price < 0 {
		errs["price"] = "tidak boleh negatif"
	}
	if r.Stock < 0 {
		errs["stock"] = "tidak boleh negatif"
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}

func (r UpdateProductRequest) Validate() error {
	if r.Name == nil && r.Price == nil && r.Stock == nil {
		return fmt.Errorf("tidak ada field yang diubah")
	}

	errs := ValidationErrors{}

	if r.Name != nil && strings.TrimSpace(*r.Name) == "" {
		errs["name"] = "tidak boleh kosong"
	}
	if r.Price != nil && *r.Price < 0 {
		errs["price"] = "tidak boleh negatif"
	}
	if r.Stock != nil && *r.Stock < 0 {
		errs["stock"] = "tidak boleh negatif"
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}
