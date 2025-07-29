package order

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"

	"github.com/sdshah09/GoCore/account"
	"github.com/sdshah09/GoCore/order/pb"
	"github.com/sdshah09/GoCore/product"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type grpcServer struct {
	pb.UnimplementedOrderServiceServer
	service       Service
	accountClient *account.Client
	productClient *product.Client
}

func ListenGRPC(service Service, accountURL string, productURL string, port int) error {
	accountClient, err := account.NewClient(accountURL)
	if err != nil {
		return err
	}
	productClient, err := product.NewClient(productURL)
	if err != nil {
		accountClient.Close()
		return err
	}
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		accountClient.Close()
		productClient.Close()
		return err
	}
	serv := grpc.NewServer()
	pb.RegisterOrderServiceServer(serv, &grpcServer{
		service:       service,
		accountClient: accountClient,
		productClient: productClient,
	})
	return serv.Serve(lis)
}

func (server *grpcServer) PostOrder(ctx context.Context, r *pb.PostOrderRequest) (*pb.PostOrderResponse, error) {
	_, err := server.accountClient.GetAccount(ctx, r.AccountID)
	if err != nil {
		log.Println("Error getting account: ", err)
		return nil, errors.New("account not found")
	}

	// Extract product IDs from request
	productIDs := []string{}
	for _, p := range r.Products {
		productIDs = append(productIDs, p.ProductId)
	}

	orderedProducts, err := server.productClient.GetProducts(ctx, "", productIDs, 0, 0)
	if err != nil {
		log.Println("Error Getting products: ", err)
		return nil, errors.New("products not found")
	}
	products := []OrderedProduct{}
	for _, p := range orderedProducts {
		product := OrderedProduct{
			ID:          p.ID,
			Quantity:    0,
			Price:       p.Price,
			Name:        p.Name,
			Description: p.Description,
		}
		for _, rp := range r.Products {
			if rp.ProductId == p.ID {
				product.Quantity = rp.Quantity
			}
		}
		if product.Quantity != 0 {
			products = append(products, product)
		}
	}
	order, err := server.service.PostOrder(ctx, r.AccountID, products)
	if err != nil {
		log.Println("Error posting order: ", err)
		return nil, errors.New("could not post order")
	}
	orderProto := &pb.Order{
		Id:         order.ID,
		AccountId:  order.AccountID,
		TotalPrice: order.TotalPrice,
		Products:   []*pb.OrderProduct{},
	}
	orderProto.CreatedAt = timestamppb.New(order.CreatedAt)
	for _, p := range order.Products {
		orderProto.Products = append(orderProto.Products, &pb.OrderProduct{
			Id:          p.ID,
			Name:        p.Name,
			Price:       p.Price,
			Description: p.Description,
			Quantity:    p.Quantity,
		})
	}
	return &pb.PostOrderResponse{
		Order: orderProto,
	}, nil
}

func (server *grpcServer) GetOrdersForAccount(ctx context.Context, r *pb.GetOrdersForAccountRequest) (*pb.GetOrdersForAccountResponse, error) {
	accountOrders, err := server.service.GetOrdersForAccount(ctx, r.AccountID)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	// Get product IDs from all orders
	productIDs := []string{}
	for _, ord := range accountOrders {
		for _, p := range ord.Products {
			productIDs = append(productIDs, p.ID)
		}
	}

	// Get product details
	products, err := server.productClient.GetProducts(ctx, "", productIDs, 0, 0)
	if err != nil {
		log.Println("Error getting products: ", err)
		return nil, err
	}

	// Create product map for quick lookup
	productMap := map[string]*product.Product{}
	for _, p := range products {
		productMap[p.ID] = &p
	}

	// Convert orders to protobuf format
	orders := []*pb.Order{}
	for _, o := range accountOrders {
		op := &pb.Order{
			AccountId:  o.AccountID,
			Id:         o.ID,
			TotalPrice: o.TotalPrice,
			Products:   []*pb.OrderProduct{},
		}
		op.CreatedAt = timestamppb.New(o.CreatedAt)

		// Add products to order
		for _, orderProduct := range o.Products {
			if product, exists := productMap[orderProduct.ID]; exists {
				op.Products = append(op.Products, &pb.OrderProduct{
					Id:          orderProduct.ID,
					Name:        product.Name,
					Description: product.Description,
					Price:       product.Price,
					Quantity:    orderProduct.Quantity,
				})
			}
		}

		orders = append(orders, op)
	}
	return &pb.GetOrdersForAccountResponse{Orders: orders}, nil
}
