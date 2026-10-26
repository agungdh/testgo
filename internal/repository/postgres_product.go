package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"testgo/internal/models"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type postgresProductRepository struct {
	db *bun.DB
}

// NewPostgresProductRepository membuat repository produk yang tersimpan di PostgreSQL.
func NewPostgresProductRepository(db *bun.DB) ProductRepository {
	return &postgresProductRepository{db: db}
}

func (r *postgresProductRepository) List(ctx context.Context) ([]models.Product, error) {
	products := make([]models.Product, 0)

	if err := r.db.NewSelect().
		Model(&products).
		Order("created_at ASC").
		Scan(ctx); err != nil {
		return nil, err
	}

	return products, nil
}

func (r *postgresProductRepository) GetByID(ctx context.Context, id string) (models.Product, error) {
	if !isUUID(id) {
		return models.Product{}, ErrNotFound
	}

	var product models.Product

	err := r.db.NewSelect().
		Model(&product).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Product{}, ErrNotFound
		}
		return models.Product{}, err
	}

	return product, nil
}

func (r *postgresProductRepository) Create(ctx context.Context, req models.CreateProductRequest) (models.Product, error) {
	now := time.Now().UTC()
	product := models.Product{
		ID:        uuid.NewString(),
		Name:      req.Name,
		Price:     req.Price,
		Stock:     req.Stock,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if _, err := r.db.NewInsert().
		Model(&product).
		Returning("*").
		Exec(ctx); err != nil {
		return models.Product{}, err
	}

	return product, nil
}

func (r *postgresProductRepository) Update(ctx context.Context, id string, req models.UpdateProductRequest) (models.Product, error) {
	product, err := r.GetByID(ctx, id)
	if err != nil {
		return models.Product{}, err
	}

	if req.Name != nil {
		product.Name = *req.Name
	}
	if req.Price != nil {
		product.Price = *req.Price
	}
	if req.Stock != nil {
		product.Stock = *req.Stock
	}
	product.UpdatedAt = time.Now().UTC()

	if _, err := r.db.NewUpdate().
		Model(&product).
		WherePK().
		Returning("*").
		Exec(ctx); err != nil {
		return models.Product{}, err
	}

	return product, nil
}

func (r *postgresProductRepository) Delete(ctx context.Context, id string) error {
	if !isUUID(id) {
		return ErrNotFound
	}

	result, err := r.db.NewDelete().
		Model((*models.Product)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}

	return nil
}

func isUUID(id string) bool {
	_, err := uuid.Parse(id)
	return err == nil
}
