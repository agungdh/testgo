-- +goose Up
CREATE TABLE IF NOT EXISTS products (
    id          bigint      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    uuid        uuid        NOT NULL,

    name        text        NOT NULL,
    price       numeric(12, 2) NOT NULL DEFAULT 0,
    stock       integer     NOT NULL DEFAULT 0,

    created_by  bigint,
    created_at  timestamptz,
    updated_by  bigint,
    updated_at  timestamptz,
    deleted_by  bigint,
    deleted_at  timestamptz,

    CONSTRAINT products_price_non_negative CHECK (price >= 0),
    CONSTRAINT products_stock_non_negative CHECK (stock >= 0)
);

-- +goose Down
DROP TABLE IF EXISTS products;
