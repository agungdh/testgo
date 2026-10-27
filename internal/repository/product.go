package repository

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"testgo/internal/models"

	"github.com/google/uuid"
)

var (
	ErrNotFound      = errors.New("data tidak ditemukan")
	ErrInvalidCursor = errors.New("cursor tidak valid")
)

const (
	// DefaultLimit dipakai bila klien tidak mengirim parameter limit.
	DefaultLimit = 20
	// MaxLimit adalah batas atas limit yang diizinkan.
	MaxLimit = 100
)

// ListParams berisi parameter cursor pagination, sort, dan filter produk.
// Sort hanya menerima kolom bisnis; string kosong berarti id DESC.
type ListParams struct {
	After    string
	Limit    int
	Sort     string
	Name     string
	StockMin *int
	PriceMin *float64
	PriceMax *float64
}

// Page adalah satu halaman hasil list beserta penanda halaman berikutnya.
type Page struct {
	Items      []models.Product
	NextCursor *string
	HasMore    bool
}

// ProductRepository abstraksi penyimpanan produk.
// Implementasi PostgreSQL dipakai aplikasi, sedangkan versi in-memory
// dipertahankan untuk test yang tidak membutuhkan database.
type ProductRepository interface {
	List(ctx context.Context, params ListParams) (Page, error)
	GetByUUID(ctx context.Context, uuid string) (models.Product, error)
	Create(ctx context.Context, req models.CreateProductRequest) (models.Product, error)
	Update(ctx context.Context, uuid string, req models.UpdateProductRequest) (models.Product, error)
	Delete(ctx context.Context, uuid string) error
}

// sortSpec adalah hasil normalisasi parameter sort.
type sortSpec struct {
	column string
	desc   bool
}

func parseSort(raw string) sortSpec {
	desc := strings.HasPrefix(raw, "-")
	column := strings.TrimPrefix(raw, "-")

	switch column {
	case "name", "price", "stock":
		return sortSpec{column: column, desc: desc}
	default:
		return sortSpec{column: "id", desc: true}
	}
}

func (s sortSpec) less(a, b models.Product) bool {
	switch s.column {
	case "name":
		if a.Name != b.Name {
			if s.desc {
				return a.Name > b.Name
			}
			return a.Name < b.Name
		}
	case "price":
		if a.Price != b.Price {
			if s.desc {
				return a.Price > b.Price
			}
			return a.Price < b.Price
		}
	case "stock":
		if a.Stock != b.Stock {
			if s.desc {
				return a.Stock > b.Stock
			}
			return a.Stock < b.Stock
		}
	}

	if a.ID == b.ID {
		return false
	}
	if s.desc {
		return a.ID > b.ID
	}
	return a.ID < b.ID
}

func isUUIDString(raw string) bool {
	_, err := uuid.Parse(raw)
	return err == nil
}

func matchesFilter(p models.Product, params ListParams) bool {
	if params.Name != "" && !strings.Contains(strings.ToLower(p.Name), strings.ToLower(params.Name)) {
		return false
	}
	if params.StockMin != nil && p.Stock < *params.StockMin {
		return false
	}
	if params.PriceMin != nil && p.Price < *params.PriceMin {
		return false
	}
	if params.PriceMax != nil && p.Price > *params.PriceMax {
		return false
	}
	return true
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return DefaultLimit
	}
	if limit > MaxLimit {
		return MaxLimit
	}
	return limit
}

type memoryProductRepository struct {
	mu       sync.RWMutex
	nextID   int64
	products map[string]models.Product
}

func NewMemoryProductRepository() ProductRepository {
	return &memoryProductRepository{
		products: make(map[string]models.Product),
	}
}

func (r *memoryProductRepository) List(_ context.Context, params ListParams) (Page, error) {
	limit := normalizeLimit(params.Limit)
	spec := parseSort(params.Sort)

	r.mu.RLock()
	defer r.mu.RUnlock()

	var cursor *models.Product
	if params.After != "" {
		found, ok := r.findActiveLocked(params.After)
		if !ok {
			return Page{}, ErrInvalidCursor
		}
		cursor = &found
	}

	items := make([]models.Product, 0, len(r.products))
	for _, p := range r.products {
		if p.DeletedAt != nil || !matchesFilter(p, params) {
			continue
		}
		items = append(items, p)
	}

	sort.Slice(items, func(i, j int) bool {
		return spec.less(items[i], items[j])
	})

	if cursor != nil {
		filtered := items[:0]
		for _, p := range items {
			if spec.less(*cursor, p) {
				filtered = append(filtered, p)
			}
		}
		items = filtered
	}

	page := Page{Items: items}
	if len(items) > limit {
		page.HasMore = true
		page.Items = items[:limit]
		next := items[limit-1].UUID
		page.NextCursor = &next
	}

	return page, nil
}

func (r *memoryProductRepository) GetByUUID(_ context.Context, raw string) (models.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.findActiveLocked(raw)
	if !ok {
		return models.Product{}, ErrNotFound
	}
	return p, nil
}

func (r *memoryProductRepository) Create(_ context.Context, req models.CreateProductRequest) (models.Product, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nextID++
	now := time.Now().UTC()
	p := models.Product{
		ID:        r.nextID,
		UUID:      uuid.NewString(),
		Name:      req.Name,
		Price:     req.Price,
		Stock:     req.Stock,
		CreatedAt: &now,
		UpdatedAt: &now,
	}

	r.products[p.UUID] = p
	return p, nil
}

func (r *memoryProductRepository) Update(_ context.Context, raw string, req models.UpdateProductRequest) (models.Product, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.findActiveLocked(raw)
	if !ok {
		return models.Product{}, ErrNotFound
	}

	if req.Name != nil {
		p.Name = *req.Name
	}
	if req.Price != nil {
		p.Price = *req.Price
	}
	if req.Stock != nil {
		p.Stock = *req.Stock
	}
	now := time.Now().UTC()
	p.UpdatedAt = &now

	r.products[p.UUID] = p
	return p, nil
}

func (r *memoryProductRepository) Delete(_ context.Context, raw string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.findActiveLocked(raw)
	if !ok {
		return ErrNotFound
	}

	now := time.Now().UTC()
	p.DeletedAt = &now
	r.products[p.UUID] = p
	return nil
}

func (r *memoryProductRepository) findActiveLocked(raw string) (models.Product, bool) {
	if !isUUIDString(raw) {
		return models.Product{}, false
	}

	p, ok := r.products[raw]
	if !ok || p.DeletedAt != nil {
		return models.Product{}, false
	}
	return p, true
}
