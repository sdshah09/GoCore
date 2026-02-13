package main

import (
	"context"
	"log"
	"net/http"
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

	// Start HTTP health check server
	go func() {
		http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})
		http.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
			// Check Elasticsearch connectivity
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := repo.Ping(ctx); err != nil {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte("Elasticsearch not available"))
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})
		log.Println("Health check server listening on :8080")
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()

	log.Println("Listening on Port 8082...")
	service := product.NewService(repo)
	log.Fatal(product.ListenGRPC(service, 8082))

}
