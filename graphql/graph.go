// Main file handler to the GraphQL Gateway
package main

// import "github.com/99designs/gqlgen/graphql"


type Server struct {
	// accountClient *account.Client
	// productClient *product.Client
	// orderClient *order.Client
}

// *Server pointer means we return reference because it is cheap rather than cerating instance and then returning it
func NewGraphQLServer(accountUrl, productUrl, orderUrl string)  (*Server, error) {
	// accountClient, err := account.NewClient(accountUrl)
	// if err != nil {
	// 	return nil, err
	// }

	// productClient, err := product.NewClient(productUrl)
	// if err != nil {
	// 	accountClient.close()
	// 	return nil, err
	// }

	// orderClient, err := order.NewClient(orderUrl)
	// if err != nil {
	// 	accountClient.close()
	// 	productClient.close()
	// 	return nil, err
	// }

	return &Server{
		// accountClient,
		// productClient,
		// orderClient,
	}, nil
}

// func (s *Server) Mutation() MutationResolver {
// 	return  &mutationResolver{
// 		server: s,
// 	}
// }

// func (s *Server) Query() QueryResolver {
// 	return &queryResolver{
// 		server: s,
// 	}
// }

// func (s *Server) Account() AccountResolver {
// 	return &accountResolver{
// 		server: s,
// 	}
// }

func (s *Server) ToExecutableSchema() graphql.ExecutableSchema {
	return NewExecutableSchema(Config{
		Resolvers: s,
	})
}