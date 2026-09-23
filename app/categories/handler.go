package categories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/mytheresa/go-hiring-challenge/app/api"
	"github.com/mytheresa/go-hiring-challenge/models"
)

// Match the VARCHAR sizes of the categories table so oversized values are
// rejected with a 400 instead of failing in the DB with a 500.
const (
	maxCodeLength = 32
	maxNameLength = 256
)

// A create body only carries a code and a name, bounded by the lengths above.
// 4 KiB leaves room for multi-byte UTF-8 and JSON overhead while stopping
// clients from streaming huge bodies into the decoder.
const maxCreateBodyBytes = 4 << 10

type CategoryStore interface {
	GetAllCategories(ctx context.Context) ([]models.Category, error)
	CreateCategory(ctx context.Context, c *models.Category) error
}

type Response struct {
	Categories []Category `json:"categories"`
}

type Category struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type Handler struct {
	store CategoryStore
}

func NewHandler(s CategoryStore) *Handler {
	return &Handler{
		store: s,
	}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	res, err := h.store.GetAllCategories(r.Context())
	if err != nil {
		api.InternalErrorResponse(w, err)
		return
	}

	categories := make([]Category, len(res))
	for i, c := range res {
		categories[i] = Category{Code: c.Code, Name: c.Name}
	}

	api.OKResponse(w, Response{Categories: categories})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req Category
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxCreateBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		api.ErrorResponse(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	c := models.Category{
		Code: strings.TrimSpace(req.Code),
		Name: strings.TrimSpace(req.Name),
	}
	if err := validateCategory(c); err != nil {
		api.ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	err := h.store.CreateCategory(r.Context(), &c)
	if errors.Is(err, models.ErrCategoryCodeExists) {
		api.ErrorResponse(w, http.StatusConflict, "category code already exists")
		return
	}
	if err != nil {
		api.InternalErrorResponse(w, err)
		return
	}

	api.JSONResponse(w, http.StatusCreated, Category{Code: c.Code, Name: c.Name})
}

func validateCategory(c models.Category) error {
	switch {
	case c.Code == "" || c.Name == "":
		return errors.New("code and name are required")
	case utf8.RuneCountInString(c.Code) > maxCodeLength:
		return fmt.Errorf("code must be at most %d characters", maxCodeLength)
	case utf8.RuneCountInString(c.Name) > maxNameLength:
		return fmt.Errorf("name must be at most %d characters", maxNameLength)
	}
	return nil
}
