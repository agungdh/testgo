package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"testgo/internal/models"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// likeEscaper membuat karakter wildcard pada filter name diperlakukan literal.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

type postgresProductRepository struct {
	db *bun.DB
}

// NewPostgresProductRepository membuat repository produk yang tersimpan di PostgreSQL.
func NewPostgresProductRepository(db *bun.DB) ProductRepository {
	return &postgresProductRepository{db: db}
}

func (r *postgresProductRepository) List(ctx context.Context, params ListParams) (Page, error) {
	limit := normalizeLimit(params.Limit)
	spec := parseSort(params.Sort)

	products := make([]models.Product, 0, limit+1)

	q := r.db.NewSelect().
		Model(&products).
		Where("deleted_at IS NULL")

	if params.Name != "" {
		q = q.Where("name ILIKE ?", "%"+likeEscaper.Replace(params.Name)+"%")
	}
	if params.StockMin != nil {
		q = q.Where("stock >= ?", *params.StockMin)
	}
	if params.PriceMin != nil {
		q = q.Where("price >= ?", *params.PriceMin)
	}
	if params.PriceMax != nil {
		q = q.Where("price <= ?", *params.PriceMax)
	}

	if params.After != "" {
		value, cursorID, err := r.resolveCursor(ctx, params.After, spec)
		if err != nil {
			return Page{}, err
		}

		if spec.column == "id" {
			q = q.Where("id < ?", cursorID)
		} else {
			operator := ">"
			if spec.desc {
				operator = "<"
			}
			q = q.Where("("+spec.column+", id) "+operator+" (?, ?)", value, cursorID)
		}
	}

	direction := "ASC"
	if spec.desc {
		direction = "DESC"
	}

	q = q.OrderExpr(spec.column + " " + direction)
	if spec.column != "id" {
		// uuid v4 acak, jadi id dipakai sebagai tiebreaker yang stabil.
		q = q.OrderExpr("id " + direction)
	}
	q = q.Limit(int64(limit + 1))

	if err := q.Scan(ctx); err != nil {
		return Page{}, err
	}

	page := Page{Items: products}
	if len(products) > limit {
		page.HasMore = true
		page.Items = products[:limit]
		next := products[limit-1].UUID
		page.NextCursor = &next
	}

	return page, nil
}

func (r *postgresProductRepository) GetByUUID(ctx context.Context, raw string) (models.Product, error) {
	if !isUUIDString(raw) {
		return models.Product{}, ErrNotFound
	}

	var product models.Product

	err := r.db.NewSelect().
		Model(&product).
		Where("uuid = ?", raw).
		Where("deleted_at IS NULL").
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
		UUID:      uuid.NewString(),
		Name:      req.Name,
		Price:     req.Price,
		Stock:     req.Stock,
		CreatedAt: &now,
		UpdatedAt: &now,
	}

	if _, err := r.db.NewInsert().
		Model(&product).
		Returning("*").
		Exec(ctx); err != nil {
		return models.Product{}, err
	}

	return product, nil
}

func (r *postgresProductRepository) Update(ctx context.Context, raw string, req models.UpdateProductRequest) (models.Product, error) {
	if !isUUIDString(raw) {
		return models.Product{}, ErrNotFound
	}

	q := r.db.NewUpdate().
		Model((*models.Product)(nil)).
		Set("updated_at = ?", time.Now().UTC()).
		Where("uuid = ?", raw).
		Where("deleted_at IS NULL").
		Returning("*")

	if req.Name != nil {
		q = q.Set("name = ?", *req.Name)
	}
	if req.Price != nil {
		q = q.Set("price = ?", *req.Price)
	}
	if req.Stock != nil {
		q = q.Set("stock = ?", *req.Stock)
	}

	var product models.Product
	if err := q.Scan(ctx, &product); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Product{}, ErrNotFound
		}
		return models.Product{}, err
	}

	return product, nil
}

func (r *postgresProductRepository) Delete(ctx context.Context, raw string) error {
	if !isUUIDString(raw) {
		return ErrNotFound
	}

	result, err := r.db.NewUpdate().
		Model((*models.Product)(nil)).
		Set("deleted_at = ?", time.Now().UTC()).
		Where("uuid = ?", raw).
		Where("deleted_at IS NULL").
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

// resolveCursor menerjemahkan uuid cursor menjadi nilai kolom sort + id internal.
func (r *postgresProductRepository) resolveCursor(ctx context.Context, raw string, spec sortSpec) (any, int64, error) {
	if !isUUIDString(raw) {
		return nil, 0, ErrInvalidCursor
	}

	var row models.Product

	err := r.db.NewSelect().
		Model(&row).
		Column("id", "name", "price", "stock").
		Where("uuid = ?", raw).
		Where("deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, 0, ErrInvalidCursor
		}
		return nil, 0, err
	}

	switch spec.column {
	case "name":
		return row.Name, row.ID, nil
	case "price":
		return row.Price, row.ID, nil
	case "stock":
		return row.Stock, row.ID, nil
	default:
		return nil, row.ID, nil
	}
}
