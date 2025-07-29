package product

import (
	"context"

	"github.com/sdshah09/GoCore/product/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn *grpc.ClientConn
	service pb.ProductServiceClient
}

func NewClient(url string) (*Client, error) {
	conn, err := grpc.NewClient(url, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	service := pb.NewProductServiceClient(conn)
	return &Client{
		conn:    conn,
		service: service,
	}, nil
}

func (c *Client) Close() {
	c.conn.Close()
}

func (client *Client) PostProduct(ctx context.Context, name string, description string, price float64) (*Product, error) {
	res, err := client.service.PostProduct(ctx, &pb.PostProductRequest{
		Name: name,
		Description: description,
		Price: price,
	},)
	if err != nil {
		return nil, err
	}
	return &Product{
		ID: res.Product.Id,
		Name: res.Product.Name,
		Description: res.Product.Description,
		Price: res.Product.Price,
	}, nil
}

func (client *Client) GetProduct(ctx context.Context, id string) (*Product, error) {
	res, err := client.service.GetProduct(ctx, &pb.GetProductRequest{Id: id})
	if err != nil {
		return nil, err
	}
	return &Product{
		ID: res.Product.Id,
		Name: res.Product.Name,
		Description: res.Product.Description,
		Price: res.Product.Price,
	}, nil
}

func (client *Client) GetProducts(ctx context.Context, query string, ids []string, skip uint64, take uint64) ([]Product, error) {
	res, err := client.service.GetProducts(ctx, &pb.GetProductsRequest{
		Skip: skip,
		Take: take,
		Query: query,
		Ids: ids,
	})
	if err != nil {
		return nil, err
	}
	var products []Product
	for _, pbProduct := range res.Products {
		products = append(products, Product{
			ID: pbProduct.Id,
			Name: pbProduct.Name,
			Description: pbProduct.Description,
			Price: pbProduct.Price,
		})
	}
	return products, nil
}