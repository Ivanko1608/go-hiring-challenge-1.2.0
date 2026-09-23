package catalog

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/shopspring/decimal"

	"github.com/mytheresa/go-hiring-challenge/app/api"
	"github.com/mytheresa/go-hiring-challenge/models"
)

const (
	defaultLimit = 10
	minLimit     = 1
	maxLimit     = 100
)

type ProductStore interface {
	GetProductsFiltered(ctx context.Context, f models.ProductFilter) ([]models.Product, int64, error)
	GetProductByCode(ctx context.Context, code string) (models.Product, error)
}

type Response struct {
	Products []Product `json:"products"`
	Total    int64     `json:"total"`
}

type Category struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type Product struct {
	Code string `json:"code"`
	// UNSAFE: money must never go through float64, it cannot represent most
	// decimal amounts exactly. Prices should be serialised as decimal strings.
	// Kept as float64 (here and in Variant) only to preserve the existing
	// response contract.
	Price    float64  `json:"price"`
	Category Category `json:"category"`
}

type ProductDetails struct {
	Product
	Variants []Variant `json:"variants"`
}

type Variant struct {
	Name  string  `json:"name"`
	SKU   string  `json:"sku"`
	Price float64 `json:"price"`
}

type Handler struct {
	store ProductStore
}

func NewHandler(s ProductStore) *Handler {
	return &Handler{
		store: s,
	}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	filter, err := parseFilter(r)
	if err != nil {
		api.ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	res, total, err := h.store.GetProductsFiltered(r.Context(), filter)
	if err != nil {
		api.InternalErrorResponse(w, err)
		return
	}

	products := make([]Product, len(res))
	for i, p := range res {
		products[i] = newProduct(p)
	}

	api.OKResponse(w, Response{
		Products: products,
		Total:    total,
	})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	p, err := h.store.GetProductByCode(r.Context(), r.PathValue("code"))
	if errors.Is(err, models.ErrProductNotFound) {
		api.ErrorResponse(w, http.StatusNotFound, "product not found")
		return
	}
	if err != nil {
		api.InternalErrorResponse(w, err)
		return
	}

	variants := make([]Variant, len(p.Variants))
	for i, v := range p.Variants {
		price := p.Price
		if v.Price.Valid {
			price = v.Price.Decimal
		}
		variants[i] = Variant{
			Name:  v.Name,
			SKU:   v.SKU,
			Price: price.InexactFloat64(),
		}
	}

	api.OKResponse(w, ProductDetails{
		Product:  newProduct(p),
		Variants: variants,
	})
}

func parseFilter(r *http.Request) (models.ProductFilter, error) {
	q := r.URL.Query()
	f := models.ProductFilter{
		Limit:        defaultLimit,
		CategoryCode: q.Get("category"),
	}

	if s := q.Get("offset"); s != "" {
		offset, err := strconv.Atoi(s)
		if err != nil || offset < 0 {
			return f, errors.New("offset must be a non-negative integer")
		}
		f.Offset = offset
	}

	if s := q.Get("limit"); s != "" {
		limit, err := strconv.Atoi(s)
		if err != nil {
			return f, errors.New("limit must be an integer")
		}
		f.Limit = min(max(limit, minLimit), maxLimit)
	}

	if s := q.Get("price_lt"); s != "" {
		price, err := decimal.NewFromString(s)
		if err != nil {
			return f, errors.New("price_lt must be a number")
		}
		f.PriceLessThan = decimal.NewNullDecimal(price)
	}

	return f, nil
}

func newProduct(p models.Product) Product {
	return Product{
		Code:  p.Code,
		Price: p.Price.InexactFloat64(),
		Category: Category{
			Code: p.Category.Code,
			Name: p.Category.Name,
		},
	}
}
