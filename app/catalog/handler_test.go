package catalog

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"github.com/mytheresa/go-hiring-challenge/models"
)

type fakeProductStore struct {
	products  []models.Product
	total     int64
	product   models.Product
	err       error
	calls     int
	gotFilter models.ProductFilter
	gotCode   string
}

func (s *fakeProductStore) GetProductsFiltered(_ context.Context, f models.ProductFilter) ([]models.Product, int64, error) {
	s.calls++
	s.gotFilter = f
	return s.products, s.total, s.err
}

func (s *fakeProductStore) GetProductByCode(_ context.Context, code string) (models.Product, error) {
	s.calls++
	s.gotCode = code
	return s.product, s.err
}

var clothing = models.Category{ID: 1, Code: "clothing", Name: "Clothing"}

func TestHandlerList(t *testing.T) {
	store := &fakeProductStore{
		products: []models.Product{
			{Code: "PROD001", Price: decimal.RequireFromString("10.99"), Category: clothing},
			{Code: "PROD004", Price: decimal.RequireFromString("15.00"), Category: clothing},
		},
		total: 8,
	}

	recorder := httptest.NewRecorder()
	NewHandler(store).List(recorder, httptest.NewRequest(http.MethodGet, "/catalog", nil))

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `{
		"products": [
			{"code": "PROD001", "price": 10.99, "category": {"code": "clothing", "name": "Clothing"}},
			{"code": "PROD004", "price": 15, "category": {"code": "clothing", "name": "Clothing"}}
		],
		"total": 8
	}`, recorder.Body.String())
}

func TestHandlerListQueryParameters(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		wantFilter models.ProductFilter
	}{
		{
			name:       "defaults",
			query:      "",
			wantFilter: models.ProductFilter{Offset: 0, Limit: 10},
		},
		{
			name:       "offset and limit",
			query:      "?offset=20&limit=5",
			wantFilter: models.ProductFilter{Offset: 20, Limit: 5},
		},
		{
			name:       "limit above maximum is clamped to 100",
			query:      "?limit=500",
			wantFilter: models.ProductFilter{Limit: 100},
		},
		{
			name:       "limit below minimum is clamped to 1",
			query:      "?limit=0",
			wantFilter: models.ProductFilter{Limit: 1},
		},
		{
			name:       "category filter",
			query:      "?category=shoes",
			wantFilter: models.ProductFilter{Limit: 10, CategoryCode: "shoes"},
		},
		{
			name:  "price less than filter",
			query: "?price_lt=12.50",
			wantFilter: models.ProductFilter{
				Limit:         10,
				PriceLessThan: decimal.NewNullDecimal(decimal.RequireFromString("12.50")),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeProductStore{}

			recorder := httptest.NewRecorder()
			NewHandler(store).List(recorder, httptest.NewRequest(http.MethodGet, "/catalog"+tt.query, nil))

			assert.Equal(t, http.StatusOK, recorder.Code)
			assert.Equal(t, tt.wantFilter, store.gotFilter)
		})
	}
}

func TestHandlerListInvalidQueryParameters(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		wantError string
	}{
		{name: "negative offset", query: "?offset=-1", wantError: "offset must be a non-negative integer"},
		{name: "non-numeric offset", query: "?offset=abc", wantError: "offset must be a non-negative integer"},
		{name: "non-numeric limit", query: "?limit=abc", wantError: "limit must be an integer"},
		{name: "non-numeric price", query: "?price_lt=cheap", wantError: "price_lt must be a number"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeProductStore{}

			recorder := httptest.NewRecorder()
			NewHandler(store).List(recorder, httptest.NewRequest(http.MethodGet, "/catalog"+tt.query, nil))

			assert.Equal(t, http.StatusBadRequest, recorder.Code)
			assert.JSONEq(t, `{"error":"`+tt.wantError+`"}`, recorder.Body.String())
			assert.Zero(t, store.calls)
		})
	}
}

func TestHandlerListStoreError(t *testing.T) {
	store := &fakeProductStore{err: errors.New("connection refused")}

	recorder := httptest.NewRecorder()
	NewHandler(store).List(recorder, httptest.NewRequest(http.MethodGet, "/catalog", nil))

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.JSONEq(t, `{"error":"internal server error"}`, recorder.Body.String())
}

func TestHandlerGet(t *testing.T) {
	store := &fakeProductStore{
		product: models.Product{
			Code:     "PROD001",
			Price:    decimal.RequireFromString("10.99"),
			Category: clothing,
			Variants: []models.Variant{
				{Name: "Variant A", SKU: "SKU001A", Price: decimal.NewNullDecimal(decimal.RequireFromString("11.99"))},
				{Name: "Variant B", SKU: "SKU001B"},
			},
		},
	}

	recorder := httptest.NewRecorder()
	NewHandler(store).Get(recorder, newGetRequest("PROD001"))

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "PROD001", store.gotCode)
	assert.JSONEq(t, `{
		"code": "PROD001",
		"price": 10.99,
		"category": {"code": "clothing", "name": "Clothing"},
		"variants": [
			{"name": "Variant A", "sku": "SKU001A", "price": 11.99},
			{"name": "Variant B", "sku": "SKU001B", "price": 10.99}
		]
	}`, recorder.Body.String())
}

func TestHandlerGetWithoutVariants(t *testing.T) {
	store := &fakeProductStore{
		product: models.Product{Code: "PROD006", Price: decimal.RequireFromString("5.50"), Category: clothing},
	}

	recorder := httptest.NewRecorder()
	NewHandler(store).Get(recorder, newGetRequest("PROD006"))

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `{
		"code": "PROD006",
		"price": 5.5,
		"category": {"code": "clothing", "name": "Clothing"},
		"variants": []
	}`, recorder.Body.String())
}

func TestHandlerGetNotFound(t *testing.T) {
	store := &fakeProductStore{err: models.ErrProductNotFound}

	recorder := httptest.NewRecorder()
	NewHandler(store).Get(recorder, newGetRequest("NOPE"))

	assert.Equal(t, http.StatusNotFound, recorder.Code)
	assert.JSONEq(t, `{"error":"product not found"}`, recorder.Body.String())
}

func TestHandlerGetStoreError(t *testing.T) {
	store := &fakeProductStore{err: errors.New("connection refused")}

	recorder := httptest.NewRecorder()
	NewHandler(store).Get(recorder, newGetRequest("PROD001"))

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.JSONEq(t, `{"error":"internal server error"}`, recorder.Body.String())
}

func newGetRequest(code string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/catalog/"+code, nil)
	r.SetPathValue("code", code)
	return r
}
