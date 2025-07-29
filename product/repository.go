package product

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"github.com/olivere/elastic/v7"
)

var ErrNotFound = errors.New("product not found")

type Repository interface {
	Close()
	PutProduct(ctx context.Context, product Product) error
	GetProductByID(ctx context.Context, id string) (*Product, error)
	ListAllProducts(ctx context.Context, skip uint64, take uint64) ([]Product, error)
	ListProductsWithIDs(ctx context.Context, ids []string) ([]Product, error)
	SearchProducts(ctx context.Context, query string, skip uint64, take uint64) ([]Product, error)
}

type elasticRepository struct {
	client *elastic.Client
}

type productDocument struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Price       float64 `json:"price"`
}


func NewElasticRepository(url string) (Repository, error) {
	client, err := elastic.NewClient(
		elastic.SetURL(url),
		elastic.SetSniff(false), // disables local cluster sniffing; important for remote/cloud
	)
	if err != nil {
		return nil, err
	}
	return &elasticRepository{client: client}, nil
}

func (r *elasticRepository) Close() {
}

// PUT /products/_doc/123
// Body: {"name": "iPhone", "description": "Smartphone", "price": "999.99"}
func (repo *elasticRepository) PutProduct(ctx context.Context, product Product) error {
	doc := productDocument{
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price,
	}
	_, err := repo.client.Index().
		Index("products").
		Id(product.ID).
		BodyJson(doc).
		Do(ctx)
	return err
}

// GET /products/_doc/123
// Returns: {"_id": "123", "_source": {"name": "iPhone", "price": "999.99"}}
func (repo *elasticRepository) GetProductByID(ctx context.Context, id string) (*Product, error) {
	res, err := repo.client.Get().
		Index("products").
		Id(id).
		Do(ctx)
	if err != nil {
		if elastic.IsNotFound(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	var doc productDocument
	if err := json.Unmarshal(res.Source, &doc); err != nil {
		return nil, err
	}

	return &Product{
		ID:          id,
		Name:        doc.Name,
		Description: doc.Description,
		Price:       doc.Price,
	}, nil
}

// GET /products/_search
// Body: {"query": {"match_all": {}}, "from": 0, "size": 10}
// Returns: {"hits": {"hits": [{"_id": "123", "_source": {...}}]}}
func (repo *elasticRepository) ListAllProducts(ctx context.Context, skip uint64, take uint64) ([]Product, error) {
	res, err := repo.client.Search().
		Index("products").
		Query(elastic.NewMatchAllQuery()).
		From(int(skip)).Size(int(take)).
		Do(ctx)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	products := []Product{}
	for _, hit := range res.Hits.Hits {
		p := productDocument{}
		if err = json.Unmarshal(hit.Source, &p); err == nil {
			products = append(products, Product{
				ID:          hit.Id,
				Name:        p.Name,
				Description: p.Description,
				Price:       p.Price,
			})
		}
	}
	return products, nil
}

// GET /products/_mget
// Body: {"docs": [{"_id": "123"}, {"_id": "456"}]}
// Returns: {"docs": [{"_id": "123", "_source": {...}}, {"_id": "456", "_source": {...}}]}
func (repo *elasticRepository) ListProductsWithIDs(ctx context.Context, ids []string) ([]Product, error) {
	items := []*elastic.MultiGetItem{}
	for _, id := range ids {
		items = append(
			items,
			elastic.NewMultiGetItem().
				Index("products").
				Id(id),
		)
	}
	res, err := repo.client.MultiGet().
		Add(items...).
		Do(ctx)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	products := []Product{}
	for _, doc := range res.Docs {
		p := productDocument{}
		if err = json.Unmarshal(doc.Source, &p); err == nil {
			products = append(products, Product{
				ID:          doc.Id,
				Name:        p.Name,
				Description: p.Description,
				Price:       p.Price,
			})
		}
	}
	return products, nil
}

// GET /products/_search
// Body: {"query": {"multi_match": {"query": "phone", "fields": ["name", "description"]}}, "from": 0, "size": 10}
// Returns: {"hits": {"hits": [{"_id": "123", "_source": {"name": "iPhone"}}]}}
func (repo *elasticRepository) SearchProducts(ctx context.Context, query string, skip uint64, take uint64) ([]Product, error) {
	res, err := repo.client.Search().
		Index("products").
		Query(elastic.NewMultiMatchQuery(query, "name", "description")).
		From(int(skip)).Size(int(take)).
		Do(ctx)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	products := []Product{}
	for _, hit := range res.Hits.Hits {
		p := productDocument{}
		if err = json.Unmarshal(hit.Source, &p); err == nil {
			products = append(products, Product{
				ID:          hit.Id,
				Name:        p.Name,
				Description: p.Description,
				Price:       p.Price,
			})
		}
	}
	return products, err
}
