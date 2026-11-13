package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"example.com/game-key-marketplace/api/internal/products"
)

type fakeStore struct {
	calls int
	slug  string
	query products.Query
	err   error
	panic bool
}

func (f *fakeStore) List(ctx context.Context, q products.Query) (products.Response, error) {
	f.calls++
	f.query = q
	if _, ok := ctx.Deadline(); !ok {
		panic("missing deadline")
	}
	if f.panic {
		panic("private details")
	}
	return products.Response{Data: []products.Product{}, Pagination: products.Pagination{Page: q.Page, PageSize: q.PageSize}, Facets: products.Facets{Platform: []products.Facet{}, Region: []products.Facet{}, Genre: []products.Facet{}}}, f.err
}

type fakeDB struct{ err error }

func (f fakeDB) Ping(context.Context) error { return f.err }
func TestProductsHTTP(t *testing.T) {
	cases := []struct {
		name, method, path string
		err                error
		panic              bool
		status, calls      int
		code               string
	}{
		{"detail", "GET", "/api/products/demo-game-001", nil, false, 200, 1, ""},
		{"unknown product", "GET", "/api/products/missing", products.ErrNotFound, false, 404, 1, "not_found"},
		{"detail failure", "GET", "/api/products/demo-game-001", errors.New("private details"), false, 500, 1, "internal_error"},
		{"detail timeout", "GET", "/api/products/demo-game-001", context.DeadlineExceeded, false, 504, 1, "timeout"},
		{"detail method", "POST", "/api/products/demo-game-001", nil, false, 405, 0, "method_not_allowed"},
		{"suggest", "GET", "/api/search/suggest?q=cyberpnk", nil, false, 200, 1, ""},
		{"suggest missing", "GET", "/api/search/suggest", nil, false, 400, 0, "invalid_query"},
		{"suggest error", "GET", "/api/search/suggest?q=witcer", errors.New("private details"), false, 500, 1, "internal_error"},
		{"suggest timeout", "GET", "/api/search/suggest?q=witcer", context.DeadlineExceeded, false, 504, 1, "timeout"},
		{"success", "GET", "/api/products?q=hello&page=2", nil, false, 200, 1, ""},
		{"invalid", "GET", "/api/products?sort=no", nil, false, 400, 0, "invalid_query"},
		{"database failure", "GET", "/api/products", errors.New("private details"), false, 500, 1, "internal_error"},
		{"timeout", "GET", "/api/products", context.DeadlineExceeded, false, 504, 1, "timeout"},
		{"panic", "GET", "/api/products", nil, true, 500, 1, "internal_error"},
		{"method", "POST", "/api/products", nil, false, 405, 0, "method_not_allowed"},
		{"missing route", "GET", "/api/missing", nil, false, 404, 0, "not_found"},
		{"readiness failure", "GET", "/api/health/ready", nil, false, 503, 0, "unavailable"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := &fakeStore{err: tc.err, panic: tc.panic}
			rr := httptest.NewRecorder()
			NewRouter(fakeDB{errors.New("offline")}, store, nil).ServeHTTP(rr, httptest.NewRequest(tc.method, tc.path, nil))
			if rr.Code != tc.status || store.calls != tc.calls {
				t.Fatalf("status=%d calls=%d body=%s", rr.Code, store.calls, rr.Body.String())
			}
			if rr.Header().Get("Content-Type") != "application/json" || !json.Valid(rr.Body.Bytes()) {
				t.Fatal("expected JSON")
			}
			if strings.Contains(rr.Body.String(), "private details") {
				t.Fatal("leaked internal error")
			}
			if tc.code != "" {
				var result struct {
					Error errorBody `json:"error"`
				}
				if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
					t.Fatal(err)
				}
				if result.Error.Code != tc.code {
					t.Fatalf("error: %+v", result)
				}
			}
			if tc.name == "detail" {
				if store.slug != "demo-game-001" {
					t.Fatal("slug not forwarded")
				}
				var result products.DetailResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil || result.Data.Slug != store.slug || result.Data.Offers == nil || result.SimilarProducts == nil {
					t.Fatal("detail contract", err)
				}
			}
			if tc.name == "success" && (store.query.Search != "hello" || store.query.Page != 2) {
				t.Fatalf("query not forwarded: %+v", store.query)
			}
		})
	}
}

func (f *fakeStore) Suggest(ctx context.Context, q products.Query) (products.Suggestions, error) {
	_, err := f.List(ctx, q)
	return products.Suggestions{Data: []products.Suggestion{}}, err
}

func (f *fakeStore) Detail(ctx context.Context, slug string) (products.DetailResponse, error) {
	f.slug = slug
	_, err := f.List(ctx, products.Query{})
	return products.DetailResponse{Data: products.DetailProduct{Slug: slug, Offers: []products.Offer{}}, SimilarProducts: []products.SimilarProduct{}}, err
}
