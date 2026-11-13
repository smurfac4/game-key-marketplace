package httpapi

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCheckoutRejectsPriceAndOwnerTampering(t *testing.T) {
	for _, body := range []string{`{"card_number":"4242424242424242","price":0}`, `{"card_number":"4242424242424242","user_id":2}`, `{"card_number":"4242424242424242","total":1}`, `{} {}`} {
		rr := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/api/checkout", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		Commerce{}.Checkout(rr, r)
		if rr.Code != 400 {
			t.Fatal(rr.Code, rr.Body.String())
		}
	}
}
