package block

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCurrency(t *testing.T) {
	for _, tc := range []struct{ raw, want string }{{"", "EUR"}, {"currency=USD", "USD"}, {"currency=", ""}, {"currency=eur", ""}, {"currency=EUR&currency=EUR", ""}, {"q=x", ""}, {"currency=%FF", ""}} {
		got, err := Currency(tc.raw)
		if tc.want == "" {
			if err == nil {
				t.Fatal(tc.raw)
			}
		} else if err != nil || got != tc.want {
			t.Fatal(got, err)
		}
	}
}
func TestBlockErrors(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
		code   string
	}{{errors.New("private database details"), 500, "internal_error"}, {context.DeadlineExceeded, 504, "timeout"}} {
		rr := httptest.NewRecorder()
		Serve(rr, httptest.NewRequest("GET", "/api/home/deals", nil), func(context.Context, string) (Response[Card], error) { return Response[Card]{}, tc.err })
		if rr.Code != tc.status || !strings.Contains(rr.Body.String(), tc.code) || strings.Contains(rr.Body.String(), "private database details") {
			t.Fatal(rr.Code, rr.Body.String())
		}
	}
}
