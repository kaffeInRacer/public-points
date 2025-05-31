package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"online-shop/internal/domain/product"
	"online-shop/pkg/config"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

type Client struct {
	es *elasticsearch.Client
}

func NewClient(cfg *config.ElasticsearchConfig) (*Client, error) {
	es, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{cfg.URL},
		Username:  cfg.Username,
		Password:  cfg.Password,
	})
	if err != nil {
		return nil, err
	}

	return &Client{es: es}, nil
}

func (c *Client) Ping(ctx context.Context) error {
	res, err := c.es.Info()
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("elasticsearch error: %s", res.String())
	}

	return nil
}

type ProductDocument struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Price       float64  `json:"price"`
	CategoryID  string   `json:"category_id"`
	Category    string   `json:"category"`
	MerchantID  string   `json:"merchant_id"`
	Images      []string `json:"images"`
	Status      string   `json:"status"`
	CreatedAt   string   `json:"created_at"`
}

type SearchService struct {
	client *Client
}

func NewSearchService(client *Client) *SearchService {
	return &SearchService{client: client}
}

func (s *SearchService) IndexProduct(ctx context.Context, product *product.Product) error {
	doc := ProductDocument{
		ID:          product.ID,
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price,
		CategoryID:  product.CategoryID,
		MerchantID:  product.MerchantID,
		Images:      product.Images,
		Status:      string(product.Status),
		CreatedAt:   product.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	if product.Category != nil {
		doc.Category = product.Category.Name
	}

	data, err := json.Marshal(doc)
	if err != nil {
		return err
	}

	req := esapi.IndexRequest{
		Index:      "products",
		DocumentID: product.ID,
		Body:       bytes.NewReader(data),
		Refresh:    "true",
	}

	res, err := req.Do(ctx, s.client.es)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error indexing product: %s", res.String())
	}

	return nil
}

func (s *SearchService) DeleteProduct(ctx context.Context, productID string) error {
	req := esapi.DeleteRequest{
		Index:      "products",
		DocumentID: productID,
		Refresh:    "true",
	}

	res, err := req.Do(ctx, s.client.es)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() && res.StatusCode != 404 {
		return fmt.Errorf("error deleting product: %s", res.String())
	}

	return nil
}

type SearchQuery struct {
	Query      string
	CategoryID string
	MinPrice   float64
	MaxPrice   float64
	MerchantID string
	From       int
	Size       int
}

type SearchResult struct {
	Products []*ProductDocument `json:"products"`
	Total    int64              `json:"total"`
}

func (s *SearchService) SearchProducts(ctx context.Context, query SearchQuery) (*SearchResult, error) {
	var buf bytes.Buffer

	searchQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []interface{}{},
				"filter": []interface{}{
					map[string]interface{}{
						"term": map[string]interface{}{
							"status": "active",
						},
					},
				},
			},
		},
		"from": query.From,
		"size": query.Size,
		"sort": []interface{}{
			map[string]interface{}{
				"_score": map[string]interface{}{
					"order": "desc",
				},
			},
		},
	}

	boolQuery := searchQuery["query"].(map[string]interface{})["bool"].(map[string]interface{})

	// Add text search
	if query.Query != "" {
		boolQuery["must"] = append(boolQuery["must"].([]interface{}), map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":  query.Query,
				"fields": []string{"name^2", "description", "category"},
			},
		})
	} else {
		boolQuery["must"] = append(boolQuery["must"].([]interface{}), map[string]interface{}{
			"match_all": map[string]interface{}{},
		})
	}

	// Add filters
	if query.CategoryID != "" {
		boolQuery["filter"] = append(boolQuery["filter"].([]interface{}), map[string]interface{}{
			"term": map[string]interface{}{
				"category_id": query.CategoryID,
			},
		})
	}

	if query.MerchantID != "" {
		boolQuery["filter"] = append(boolQuery["filter"].([]interface{}), map[string]interface{}{
			"term": map[string]interface{}{
				"merchant_id": query.MerchantID,
			},
		})
	}

	// Price range filter
	if query.MinPrice > 0 || query.MaxPrice > 0 {
		priceRange := map[string]interface{}{}
		if query.MinPrice > 0 {
			priceRange["gte"] = query.MinPrice
		}
		if query.MaxPrice > 0 {
			priceRange["lte"] = query.MaxPrice
		}
		boolQuery["filter"] = append(boolQuery["filter"].([]interface{}), map[string]interface{}{
			"range": map[string]interface{}{
				"price": priceRange,
			},
		})
	}

	if err := json.NewEncoder(&buf).Encode(searchQuery); err != nil {
		return nil, err
	}

	res, err := s.client.es.Search(
		s.client.es.Search.WithContext(ctx),
		s.client.es.Search.WithIndex("products"),
		s.client.es.Search.WithBody(&buf),
		s.client.es.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("search error: %s", res.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, err
	}

	hits := response["hits"].(map[string]interface{})
	total := int64(hits["total"].(map[string]interface{})["value"].(float64))

	var products []*ProductDocument
	for _, hit := range hits["hits"].([]interface{}) {
		source := hit.(map[string]interface{})["_source"]
		var product ProductDocument
		sourceBytes, _ := json.Marshal(source)
		json.Unmarshal(sourceBytes, &product)
		products = append(products, &product)
	}

	return &SearchResult{
		Products: products,
		Total:    total,
	}, nil
}

func (s *SearchService) CreateIndex(ctx context.Context) error {
	mapping := `{
		"mappings": {
			"properties": {
				"id": {"type": "keyword"},
				"name": {
					"type": "text",
					"analyzer": "standard",
					"fields": {
						"keyword": {"type": "keyword"}
					}
				},
				"description": {"type": "text", "analyzer": "standard"},
				"price": {"type": "float"},
				"category_id": {"type": "keyword"},
				"category": {
					"type": "text",
					"analyzer": "standard",
					"fields": {
						"keyword": {"type": "keyword"}
					}
				},
				"merchant_id": {"type": "keyword"},
				"images": {"type": "keyword"},
				"status": {"type": "keyword"},
				"created_at": {"type": "date"}
			}
		}
	}`

	req := esapi.IndicesCreateRequest{
		Index: "products",
		Body:  strings.NewReader(mapping),
	}

	res, err := req.Do(ctx, s.client.es)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() && res.StatusCode != 400 { // 400 means index already exists
		return fmt.Errorf("error creating index: %s", res.String())
	}

	return nil
}