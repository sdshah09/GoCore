package order

import (
	"context"
	"database/sql"

	"github.com/lib/pq"
)

type Repository interface {
	Close()
	PutOrder(ctx context.Context, o Order) error
	GetOrdersForAccount(ctx context.Context, accountID string) ([]Order, error)
}

type postgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(url string) (Repository, error) {
	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return &postgresRepository{db}, nil
}

func (r *postgresRepository) Close() {
	r.db.Close()
}

func (r *postgresRepository) PutOrder(ctx context.Context, ord Order) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
		err = tx.Commit()
	}()
	_, err = tx.ExecContext(
		ctx,
		"INSERT INTO orders(id, created_at, account_id, total_price) VALUES ($1, $2, $3, $4)",
		ord.ID,
		ord.CreatedAt,
		ord.AccountID,
		ord.TotalPrice,
	)
	if err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx, pq.CopyIn("order_products", "order_id", "product_id", "quantity")) // Creating a queue and declaring copy into order_products table
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, p := range ord.Products {
		_, err = stmt.ExecContext(ctx, ord.ID, p.ID, p.Quantity) // Add the data to queue
		if err != nil {
			return err
		}
	}
	_, err = stmt.ExecContext(ctx) // Final call without data push all the data from queue
	if err != nil {
		return err
	}
	stmt.Close()
	return nil
}

func (r *postgresRepository) GetOrdersForAccount(ctx context.Context, accountId string) ([]Order, error) {
	orders := []Order{}
	order := &Order{}
	products := []OrderedProduct{}
	order_product := &OrderedProduct{}
	lastOrder := &Order{}

	rows, err := r.db.QueryContext(
		ctx,
		`SELECT
		ord.id,
		ord.created_at,
		ord.account_id,
		ord.total_price::money::numeric::float8,
		ord_prod.product_id,
		ord_prod.quantity
		FROM orders ord JOIN order_products ord_prod ON(ord.id = ord_prod.order_id)
		WHERE ord.account_id = $1
		ORDER BY ord.id`,
		accountId,
	)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		if err = rows.Scan(
			&order.ID,
			&order.CreatedAt,
			&order.AccountID,
			&order.TotalPrice,
			&order_product.ID,
			&order_product.Quantity,
		); err != nil {
			return nil, err
		}
		if lastOrder.ID != "" && lastOrder.ID != order.ID {
			newOrder := Order{
				ID:         lastOrder.ID,
				AccountID:  lastOrder.AccountID,
				CreatedAt:  lastOrder.CreatedAt,
				TotalPrice: lastOrder.TotalPrice,
				Products:   lastOrder.Products,
			}
			orders = append(orders, newOrder)
			products = []OrderedProduct{}
		}
		products = append(products, OrderedProduct{
			ID:       order_product.ID,
			Quantity: order_product.Quantity,
		})
		*lastOrder = *order
	}
	if lastOrder != nil {
		newOrder := Order{
			ID:         lastOrder.ID,
			AccountID:  lastOrder.AccountID,
			CreatedAt:  lastOrder.CreatedAt,
			TotalPrice: lastOrder.TotalPrice,
			Products:   lastOrder.Products,
		}
		orders = append(orders, newOrder)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return orders, nil

}
