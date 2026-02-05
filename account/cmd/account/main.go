package main

import (
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/sdshah09/GoCore/account"
	"github.com/tinrab/retry"
)

type Config struct {
	DBHost     string `envconfig:"DB_HOST"`
	DBPort     string `envconfig:"DB_PORT"`
	DBName     string `envconfig:"DB_NAME"`
	DBUser     string `envconfig:"DB_USER"`
	DBPassword string `envconfig:"DB_PASSWORD"`
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

	var repo account.Repository
	retry.ForeverSleep(2*time.Second, func(_ int) (err error) {
		repo, err = account.NewPostgresRepository(cfg.DatabaseURL())
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
