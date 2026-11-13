package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRequestBudget(t *testing.T) {
	for _, parentCanceled := range []bool{false, true} {
		t.Run(map[bool]string{false: "deadline cancels database wait", true: "disconnect cancels immediately"}[parentCanceled], func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if parentCanceled {
				cancel()
			}
			rr := httptest.NewRecorder()
			requestBudget(10*time.Millisecond)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				select {
				case <-r.Context().Done():
					commerceError(w, r.Context().Err())
				case <-time.After(time.Second):
					t.Fatal("request work was not canceled")
				}
			})).ServeHTTP(rr, httptest.NewRequest("GET", "/api/cart", nil).WithContext(ctx))
			if !parentCanceled && rr.Code != 504 {
				t.Fatalf("status=%d", rr.Code)
			}
		})
	}
}
