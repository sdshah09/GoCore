package order

import (
	"context"
	"time"

	"github.com/segmentio/ksuid"
)

type Order struct {
	ID         string
	CreatedAt  time.Time
	TotalPrice float64
	AccountID  string
	Products   []OrderedProduct
}

type OrderedProduct struct {
	ID          string
	Name        string
	Description string
	Price       float64
	Quantity    uint32
}

type orderService struct {
	repository Repository
}

type Service interface {
	PostOrder(ctx context.Context, accountID string, products []OrderedProduct) (*Order, error)
	GetOrdersForAccount(ctx context.Context, accountID string) ([]Order, error)
}

func NewService(repo Repository) Service {
	return &orderService{repo}
}

func (service *orderService) PostOrder(ctx context.Context, accountID string, products []OrderedProduct) (*Order, error) {
	var totalPrice float64
	for _, product := range products {
		totalPrice += product.Price * float64(product.Quantity)
	}
	order := &Order{
		ID:         ksuid.New().String(),
		CreatedAt:  time.Now(),
		TotalPrice: totalPrice,
		AccountID:  accountID,
		Products:   products,
	}
	err := service.repository.PutOrder(ctx, *order)
	if err != nil {
		return nil, err
	}
	return order, err
}

func (service *orderService) GetOrdersForAccount(ctx context.Context, accountID string) ([]Order, error) {
	return service.repository.GetOrdersForAccount(ctx, accountID)
}
