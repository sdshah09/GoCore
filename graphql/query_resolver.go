package main

import (
	"context"
	"log"
	"time"
)

type queryResolver struct {
	server *Server
}

func (r *queryResolver) Accounts(ctx context.Context, pagination *PaginationInput, id *string) ([]*Account, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if id != nil {
		account, err := r.server.accountClient.GetAccount(ctx, *id)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		return []*Account{{
			ID:   account.ID,
			Name: account.Name,
		}}, nil
	}
	skip, take := uint64(0), uint64(100)
	if pagination != nil {
		skip, take = pagination.bounds()
	}
	accounts, err := r.server.accountClient.GetAccounts(ctx, skip, take)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	var result []*Account
	for _, account := range accounts {
		result = append(result, &Account{
			ID:   account.ID,
			Name: account.Name,
		})
	}
	return result, nil
}

func (r *queryResolver) Products(ctx context.Context, pagination *PaginationInput, query *string, id *string) ([]*Product, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	// Get single product
	if id != nil {
		product, err := r.server.productClient.GetProduct(ctx, *id)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		return []*Product{{
			ID:          product.ID,
			Name:        product.Name,
			Description: product.Description,
			Price:       product.Price,
		}}, nil
	}

	skip, take := uint64(0), uint64(100)
	if pagination != nil {
		skip, take = pagination.bounds()
	}

	q := ""
	if query != nil {
		q = *query
	}
	productList, err := r.server.productClient.GetProducts(ctx, q, nil, skip, take)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	var products []*Product
	for _, p := range productList {
		products = append(products,
			&Product{
				ID:          p.ID,
				Name:        p.Name,
				Description: p.Description,
				Price:       p.Price,
			},
		)
	}

	return products, nil
}

func (r *queryResolver) OrdersForAccount(ctx context.Context, accountId string) ([]*Order, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	res, err := r.server.orderClient.GetOrdersForAccount(ctx, accountId)
	if err != nil {
		return nil, err
	}
	var orders []*Order
	for _, p := range res {
		newOrder := &Order{
			ID:         p.ID,
			CreatedAt:  p.CreatedAt,
			TotalPrice: p.TotalPrice,
		}
		var products []*OrderProduct
		for _, prod := range p.Products {
			products = append(products, &OrderProduct{
				ID:          prod.ID,
				Name:        prod.Name,
				Description: prod.Description,
				Price:       prod.Price,
				Quantity:    int(prod.Quantity),
			})
		}
		newOrder.Products = products
		orders = append(orders, newOrder)
	}
	return orders, nil
}

func (p PaginationInput) bounds() (uint64, uint64) {
	skipValue := uint64(0)
	takeValue := uint64(100)
	if p.Skip != nil {
		skipValue = uint64(*p.Skip)
	}
	if p.Take != nil {
		takeValue = uint64(*p.Take)
	}
	return skipValue, takeValue
}
