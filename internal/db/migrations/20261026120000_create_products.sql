-- +goose Up
CREATE TABLE IF NOT EXISTS products (
    id uuid PRIMARY KEY,
    name text NOT NULL,
    price numeric(12, 2) NOT NULL DEFAULT 0,
    stock integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT products_price_non_negative CHECK (price >= 0),
    CONSTRAINT products_stock_non_negative CHECK (stock >= 0)
);

CREATE INDEX IF NOT EXISTS products_created_at_idx ON products (created_at);

-- +goose Down
DROP TABLE IF EXISTS products;
