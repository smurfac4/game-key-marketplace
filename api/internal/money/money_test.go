package money

import "testing"

func TestConvert(t *testing.T) {
	for _, v := range []struct {
		n        int64
		from, to string
		want     int64
	}{{1000, "EUR", "USD", 1100}, {1000, "EUR", "GBP", 850}, {1100, "USD", "EUR", 1000}, {1, "EUR", "GBP", 1}} {
		if got := Convert(v.n, v.from, v.to); got != v.want {
			t.Fatal(got, v.want)
		}
	}
}
