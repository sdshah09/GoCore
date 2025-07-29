package main

import (
	"log"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/sdshah09/GoCore/product"
	"github.com/tinrab/retry"
)

type Config struct {
	DatabaseURL string `envconfig:"DATABASE_URL"`
}

func main() {
	var cfg Config
	err := envconfig.Process("", &cfg)
	if err != nil {
		log.Fatal(err)
	}

	var repo product.Repository
	retry.ForeverSleep(2*time.Second, func(_ int) (err error) {
		repo, err = product.NewElasticRepository(cfg.DatabaseURL)
		if err != nil {
			log.Println(err)
		}
		return
	})
	defer repo.Close()
	log.Println("Listening on Port 8081...")
	service := product.NewService(repo)
	log.Fatal(product.ListenGRPC(service, 8081))

}
