package main

import (
	"log"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/sdshah09/GoCore/account"
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

	var repo account.Repository
	retry.ForeverSleep(2*time.Second, func(_ int) (err error) {
		repo, err = account.NewPostgresRepository(cfg.DatabaseURL)
		if err != nil {
			log.Println(err)
		}
		return
	})
	defer repo.Close()
	log.Println("Listening on Port 8081...")
	service := account.NewService(repo)
	log.Fatal(account.ListenGRPC(service, 8081))

}
