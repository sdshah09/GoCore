package main

import (
	"log"
	"net/http"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/kelseyhightower/envconfig"
)

type AppConfig struct {
	AccountURL string `envconfig:"ACCOUNT_SERVICE_URL"`
	ProductURL string `envconfig:"PRODUCT_SERVICE_URL"`
	OrderURL   string `envconfig:"ORDER_SERVICE_URL"`
}

func main() {
	var cfg AppConfig
	err := envconfig.Process("", &cfg)
	if err != nil {
		log.Fatal(err)
	}

	server, err := NewGraphQLServer(cfg.AccountURL, cfg.ProductURL, cfg.OrderURL)
	if err != nil {
		log.Fatal(err)
	}
	http.Handle("/graphql", handler.NewDefaultServer(server.ToExecutableSchema()))
	http.Handle("/playground", playground.Handler("shaswat", "/graphql"))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
