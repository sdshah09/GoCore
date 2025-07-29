// Main file handler to the GraphQL Gateway
package main

import (
	"github.com/99designs/gqlgen/graphql"
	"github.com/sdshah09/GoCore/account"
	"github.com/sdshah09/GoCore/order"
	"github.com/sdshah09/GoCore/product"
)

type Server struct {
	accountClient *account.Client
	productClient *product.Client
	orderClient   *order.Client
}

// *Server pointer means we return reference because it is cheap rather than cerating instance and then returning it
func NewGraphQLServer(accountUrl, productUrl, orderUrl string) (*Server, error) {
	accountClient, err := account.NewClient(accountUrl)
	if err != nil {
		return nil, err
	}

	productClient, err := product.NewClient(productUrl)
	if err != nil {
		accountClient.Close()
		return nil, err
	}

	orderClient, err := order.NewClient(orderUrl)
	if err != nil {
		accountClient.Close()
		productClient.Close()
		return nil, err
	}

	return &Server{
		accountClient: accountClient,
		productClient: productClient,
		orderClient:   orderClient,
	}, nil
}

func (s *Server) Mutation() MutationResolver {
	return &mutationResolver{
		server: s,
	}
}

func (s *Server) Query() QueryResolver {
	return &queryResolver{
		server: s,
	}
}

func (s *Server) Account() AccountResolver {
	return &accountResolver{
		server: s,
	}
}

func (s *Server) ToExecutableSchema() graphql.ExecutableSchema {
	return NewExecutableSchema(Config{
		Resolvers: s,
	})
}
