package product

import (
	"context"

	"github.com/segmentio/ksuid"
)

type Product struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
}

type Service interface {
	PostProduct(ctx context.Context, name string, description string, price float64) (*Product, error)
	GetProduct(ctx context.Context, id string) (*Product, error)
	GetAllProducts(ctx context.Context, skip uint64, take uint64) ([]Product, error)
	GetProductsWithIds(ctx context.Context, ids []string) ([]Product, error)
	GetSearchProducts(ctx context.Context, query string, skip uint64, take uint64) ([]Product, error)
}

type productService struct {
	repository Repository
}

func NewService(repo Repository) Service {
	return &productService{repo}
}

func (service *productService) PostProduct(ctx context.Context, name string, description string, price float64) (*Product, error) {
	product := &Product{
		Name:        name,
		Description: description,
		Price:       price,
		ID:          ksuid.New().String(),
	}
	if err := service.repository.PutProduct(ctx, *product); err != nil {
		return nil, err
	}
	return product, nil
}

func (service *productService) GetProduct(ctx context.Context, id string) (*Product, error) {
	res, err := service.repository.GetProductByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return res, err
}

func (service *productService) GetAllProducts(ctx context.Context, skip uint64, take uint64) ([]Product, error) {
	res, err := service.repository.ListAllProducts(ctx, skip, take)
	return res, err
}

func (service *productService) GetProductsWithIds(ctx context.Context, ids []string) ([]Product, error) {
	res, err := service.repository.ListProductsWithIDs(ctx, ids)
	return res, err
}
func (service *productService) GetSearchProducts(ctx context.Context, query string, skip uint64, take uint64) ([]Product, error) {
	if take > 100 || (skip == 0 && take == 0) {
		take = 100
	}
	return service.repository.SearchProducts(ctx, query, skip, take)
}
