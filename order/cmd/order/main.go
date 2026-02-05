package main

import (
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/sdshah09/GoCore/order"
	"github.com/tinrab/retry"
)

type Config struct {
	DBHost      string `envconfig:"DB_HOST"`
	DBPort      string `envconfig:"DB_PORT"`
	DBName      string `envconfig:"DB_NAME"`
	DBUser      string `envconfig:"DB_USER"`
	DBPassword  string `envconfig:"DB_PASSWORD"`
	AccountURL  string `envconfig:"ACCOUNT_SERVICE_URL"`
	ProductURL  string `envconfig:"PRODUCT_SERVICE_URL"`
}

func (c Config) DatabaseURL() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.DBUser,
		url.QueryEscape(c.DBPassword),
		c.DBHost,
		c.DBPort,
		c.DBName,
	)
}

func main() {
	var cfg Config
	err := envconfig.Process("", &cfg)
	if err != nil {
		log.Fatal(err)
	}
	var repo order.Repository
	retry.ForeverSleep(2*time.Second, func(_ int) (err error) {
		repo, err = order.NewPostgresRepository(cfg.DatabaseURL())
		if err != nil {
			log.Println(err)
		}
		return
	})
	defer repo.Close()
	log.Println("Listening on 8083...")
	s := order.NewService(repo)
	log.Fatal(order.ListenGRPC(s, cfg.AccountURL, cfg.ProductURL, 8083))
}
