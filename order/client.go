package order

import (
	"context"

	"github.com/sdshah09/GoCore/order/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn    *grpc.ClientConn
	service pb.OrderServiceClient
}

func NewClient(url string) (*Client, error) {
	conn, err := grpc.NewClient(url, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	service := pb.NewOrderServiceClient(conn)
	return &Client{
		conn:    conn,
		service: service,
	}, nil
}

func (client *Client) Close() {
	client.conn.Close()
}

func (client *Client) PostOrder(ctx context.Context, accountID string, products []OrderedProduct) (*Order, error) {
	// Convert OrderProduct to protobuf OrderedProduct
	protoProducts := []*pb.PostOrderRequest_OrderedProduct{}
	for _, p := range products {
		protoProducts = append(protoProducts, &pb.PostOrderRequest_OrderedProduct{
			ProductId: p.ID,
			Quantity:  p.Quantity,
		})
	}

	response, err := client.service.PostOrder(ctx, &pb.PostOrderRequest{
		AccountID: accountID,
		Products:  protoProducts,
	})
	if err != nil {
		return nil, err
	}

	// Convert protobuf response to Order
	order := &Order{
		ID:         response.Order.Id,
		AccountID:  response.Order.AccountId,
		TotalPrice: response.Order.TotalPrice,
		CreatedAt:  response.Order.CreatedAt.AsTime(),
		Products:   []OrderedProduct{},
	}

	for _, p := range response.Order.Products {
		order.Products = append(order.Products, OrderedProduct{
			ID:          p.Id,
			Name:        p.Name,
			Description: p.Description,
			Price:       p.Price,
			Quantity:    p.Quantity,
		})
	}

	return order, nil
}

func (client *Client) GetOrdersForAccount(ctx context.Context, accountID string) ([]Order, error) {
	res, err := client.service.GetOrdersForAccount(ctx, &pb.GetOrdersForAccountRequest{AccountID: accountID})
	if err != nil {
		return nil, err
	}
	orders := []Order{}
	for _, orderProto := range res.Orders {
		newOrder := Order{
			ID:         orderProto.Id,
			TotalPrice: orderProto.TotalPrice,
			AccountID:  orderProto.AccountId,
			CreatedAt:  orderProto.CreatedAt.AsTime(),
		}
		products := []OrderedProduct{}
		for _, p := range orderProto.Products {
			products = append(products, OrderedProduct{
				ID:          p.Id,
				Quantity:    p.Quantity,
				Name:        p.Name,
				Description: p.Description,
				Price:       p.Price,
			})
		}
		newOrder.Products = products
		orders = append(orders, newOrder)
	}
	return orders, nil
}
