package product

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/sdshah09/GoCore/product/pb"
	"google.golang.org/grpc"
)

type grpcServer struct {
	pb.UnimplementedProductServiceServer
	service Service
}

func ListenGRPC(s Service, port int) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	serv := grpc.NewServer()
	pb.RegisterProductServiceServer(serv, &grpcServer{service: s})
	return serv.Serve(lis)
}

func (server *grpcServer) PostProduct(ctx context.Context, r *pb.PostProductRequest) (*pb.PostProductResponse, error) {
	product, err := server.service.PostProduct(ctx, r.Name, r.Description, r.Price)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return &pb.PostProductResponse{
		Product: &pb.Product{
			Id:          product.ID,
			Description: product.Description,
			Price:       product.Price,
			Name:        product.Name,
		},
	}, nil
}

func (server *grpcServer) GetProduct(ctx context.Context, r *pb.GetProductRequest) (*pb.GetProductResponse, error) {
	product, err := server.service.GetProduct(ctx, r.Id)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return &pb.GetProductResponse{
		Product: &pb.Product{
			Id:          product.ID,
			Description: product.Description,
			Name:        product.Name,
			Price:       product.Price,
		},
	}, nil
}

func (server *grpcServer) GetProducts(ctx context.Context, r *pb.GetProductsRequest) (*pb.GetProductsResponse, error) {
	var res []Product
	var err error
	if r.Query != "" {
		res, err = server.service.GetSearchProducts(ctx, r.Query, r.Skip, r.Take)
	} else if len(r.Ids) != 0 {
		res, err = server.service.GetProductsWithIds(ctx, r.Ids)
	} else {
		res, err = server.service.GetAllProducts(ctx, r.Skip, r.Take)
	}
	if err != nil {
		log.Println(err)
		return nil, err
	}
	products := []*pb.Product{}
	for _, p := range res {
		products = append(products, &pb.Product{
			Id:          p.ID,
			Name:        p.Name,
			Description: p.Description,
			Price:       p.Price,
		})
	}

	return &pb.GetProductsResponse{Products: products}, nil
}
