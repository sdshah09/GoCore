package main

import (
	"context"
	"log"
	"time"
)

type accountResolver struct {
	server *Server
}

func (r *accountResolver) Orders(ctx context.Context, obj *Account) ([]*Order, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	orders, err := r.server.orderClient.GetOrdersForAccount(ctx, obj.ID)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	var result []*Order
	for _, order := range orders {
		// Convert OrderProduct to *OrderProduct for GraphQL
		var orderProducts []*OrderProduct
		for _, p := range order.Products {
			orderProducts = append(orderProducts, &OrderProduct{
				ID:          p.ID,
				Name:        p.Name,
				Description: p.Description,
				Price:       p.Price,
				Quantity:    int(p.Quantity),
			})
		}

		result = append(result, &Order{
			ID:         order.ID,
			CreatedAt:  order.CreatedAt,
			TotalPrice: order.TotalPrice,
			Products:   orderProducts,
		})
	}

	return result, nil
}
