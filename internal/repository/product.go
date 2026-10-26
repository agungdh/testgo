package repository

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"testgo/internal/models"

	"github.com/google/uuid"
)

var (
	ErrNotFound = errors.New("data tidak ditemukan")
)

// ProductRepository abstraksi penyimpanan produk.
// Implementasi saat ini in-memory, mudah diganti dengan database.
type ProductRepository interface {
	List(ctx context.Context) ([]models.Product, error)
	GetByID(ctx context.Context, id string) (models.Product, error)
	Create(ctx context.Context, req models.CreateProductRequest) (models.Product, error)
	Update(ctx context.Context, id string, req models.UpdateProductRequest) (models.Product, error)
	Delete(ctx context.Context, id string) error
}

type memoryProductRepository struct {
	mu       sync.RWMutex
	products map[string]models.Product
}

func NewMemoryProductRepository() ProductRepository {
	return &memoryProductRepository{
		products: make(map[string]models.Product),
	}
}

func (r *memoryProductRepository) List(_ context.Context) ([]models.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]models.Product, 0, len(r.products))
	for _, p := range r.products {
		result = append(result, p)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})

	return result, nil
}

func (r *memoryProductRepository) GetByID(_ context.Context, id string) (models.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.products[id]
	if !ok {
		return models.Product{}, ErrNotFound
	}
	return p, nil
}

func (r *memoryProductRepository) Create(_ context.Context, req models.CreateProductRequest) (models.Product, error) {
	now := time.Now().UTC()
	p := models.Product{
		ID:        uuid.NewString(),
		Name:      req.Name,
		Price:     req.Price,
		Stock:     req.Stock,
		CreatedAt: now,
		UpdatedAt: now,
	}

	r.mu.Lock()
	r.products[p.ID] = p
	r.mu.Unlock()

	return p, nil
}

func (r *memoryProductRepository) Update(_ context.Context, id string, req models.UpdateProductRequest) (models.Product, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.products[id]
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
	p.UpdatedAt = time.Now().UTC()

	r.products[id] = p
	return p, nil
}

func (r *memoryProductRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.products[id]; !ok {
		return ErrNotFound
	}

	delete(r.products, id)
	return nil
}
