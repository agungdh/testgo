package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/uptrace/bun"
)

// Product adalah representasi produk yang disimpan dan dikembalikan via API.
// id internal dan kolom audit tidak pernah dikirim ke klien.
type Product struct {
	bun.BaseModel `bun:"table:products,alias:p"`

	ID    int64   `bun:"id,pk,autoincrement" json:"-"`
	UUID  string  `bun:"uuid,notnull,type:uuid" json:"uuid"`
	Name  string  `bun:"name,notnull" json:"name"`
	Price float64 `bun:"price,notnull,type:numeric(12,2)" json:"price"`
	Stock int     `bun:"stock,notnull" json:"stock"`

	CreatedBy *int64     `bun:"created_by" json:"-"`
	CreatedAt *time.Time `bun:"created_at" json:"-"`
	UpdatedBy *int64     `bun:"updated_by" json:"-"`
	UpdatedAt *time.Time `bun:"updated_at" json:"-"`
	DeletedBy *int64     `bun:"deleted_by" json:"-"`
	DeletedAt *time.Time `bun:"deleted_at" json:"-"`
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
