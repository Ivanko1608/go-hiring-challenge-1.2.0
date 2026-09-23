package categories

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mytheresa/go-hiring-challenge/models"
)

type fakeCategoryStore struct {
	categories  []models.Category
	err         error
	calls       int
	gotCategory models.Category
}

func (s *fakeCategoryStore) GetAllCategories(_ context.Context) ([]models.Category, error) {
	s.calls++
	return s.categories, s.err
}

func (s *fakeCategoryStore) CreateCategory(_ context.Context, c *models.Category) error {
	s.calls++
	s.gotCategory = *c
	return s.err
}

func TestHandlerList(t *testing.T) {
	store := &fakeCategoryStore{
		categories: []models.Category{
			{ID: 1, Code: "clothing", Name: "Clothing"},
			{ID: 2, Code: "shoes", Name: "Shoes"},
		},
	}

	recorder := httptest.NewRecorder()
	NewHandler(store).List(recorder, httptest.NewRequest(http.MethodGet, "/categories", nil))

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `{
		"categories": [
			{"code": "clothing", "name": "Clothing"},
			{"code": "shoes", "name": "Shoes"}
		]
	}`, recorder.Body.String())
}

func TestHandlerListEmpty(t *testing.T) {
	store := &fakeCategoryStore{}

	recorder := httptest.NewRecorder()
	NewHandler(store).List(recorder, httptest.NewRequest(http.MethodGet, "/categories", nil))

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `{"categories": []}`, recorder.Body.String())
}

func TestHandlerListStoreError(t *testing.T) {
	store := &fakeCategoryStore{err: errors.New("connection refused")}

	recorder := httptest.NewRecorder()
	NewHandler(store).List(recorder, httptest.NewRequest(http.MethodGet, "/categories", nil))

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.JSONEq(t, `{"error":"internal server error"}`, recorder.Body.String())
}

func TestHandlerCreate(t *testing.T) {
	store := &fakeCategoryStore{}

	recorder := httptest.NewRecorder()
	NewHandler(store).Create(recorder, newCreateRequest(`{"code": " bags ", "name": " Bags "}`))

	assert.Equal(t, http.StatusCreated, recorder.Code)
	assert.JSONEq(t, `{"code": "bags", "name": "Bags"}`, recorder.Body.String())
	assert.Equal(t, models.Category{Code: "bags", Name: "Bags"}, store.gotCategory)
}

func TestHandlerCreateInvalidBody(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantError string
	}{
		{name: "malformed JSON", body: `{"code":`, wantError: "invalid JSON body"},
		{name: "unknown field", body: `{"code": "bags", "name": "Bags", "id": 5}`, wantError: "invalid JSON body"},
		{name: "body too large", body: `{"code": "` + strings.Repeat("a", maxCreateBodyBytes) + `"}`, wantError: "invalid JSON body"},
		{name: "missing code", body: `{"name": "Bags"}`, wantError: "code and name are required"},
		{name: "blank name", body: `{"code": "bags", "name": "   "}`, wantError: "code and name are required"},
		{
			name:      "code too long",
			body:      `{"code": "` + strings.Repeat("a", maxCodeLength+1) + `", "name": "Bags"}`,
			wantError: "code must be at most 32 characters",
		},
		{
			name:      "name too long",
			body:      `{"code": "bags", "name": "` + strings.Repeat("a", maxNameLength+1) + `"}`,
			wantError: "name must be at most 256 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeCategoryStore{}

			recorder := httptest.NewRecorder()
			NewHandler(store).Create(recorder, newCreateRequest(tt.body))

			assert.Equal(t, http.StatusBadRequest, recorder.Code)
			assert.JSONEq(t, `{"error":"`+tt.wantError+`"}`, recorder.Body.String())
			assert.Zero(t, store.calls)
		})
	}
}

func TestHandlerCreateCountsCharactersNotBytes(t *testing.T) {
	store := &fakeCategoryStore{}
	code := strings.Repeat("é", maxCodeLength)

	recorder := httptest.NewRecorder()
	NewHandler(store).Create(recorder, newCreateRequest(`{"code": "`+code+`", "name": "Accents"}`))

	assert.Equal(t, http.StatusCreated, recorder.Code)
}

func TestHandlerCreateCodeExists(t *testing.T) {
	store := &fakeCategoryStore{err: models.ErrCategoryCodeExists}

	recorder := httptest.NewRecorder()
	NewHandler(store).Create(recorder, newCreateRequest(`{"code": "shoes", "name": "Shoes"}`))

	assert.Equal(t, http.StatusConflict, recorder.Code)
	assert.JSONEq(t, `{"error":"category code already exists"}`, recorder.Body.String())
}

func TestHandlerCreateStoreError(t *testing.T) {
	store := &fakeCategoryStore{err: errors.New("connection refused")}

	recorder := httptest.NewRecorder()
	NewHandler(store).Create(recorder, newCreateRequest(`{"code": "bags", "name": "Bags"}`))

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.JSONEq(t, `{"error":"internal server error"}`, recorder.Body.String())
}

func newCreateRequest(body string) *http.Request {
	return httptest.NewRequest(http.MethodPost, "/categories", strings.NewReader(body))
}
