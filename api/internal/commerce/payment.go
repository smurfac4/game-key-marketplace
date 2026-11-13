package commerce

import (
	"context"
	"errors"
	"strings"
)

var ErrPaymentDeclined = errors.New("mock payment declined")
var ErrInvalidCard = errors.New("use a supported mock card number")

type Payment struct {
	OrderID, AmountCents int64
	Currency, CardNumber string
}
type PaymentProvider interface {
	Charge(context.Context, Payment) error
}
type MockPayment struct{}

func (MockPayment) Charge(ctx context.Context, p Payment) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	switch strings.ReplaceAll(strings.ReplaceAll(p.CardNumber, " ", ""), "-", "") {
	case "4242424242424242":
		return nil
	case "4000000000000002":
		return ErrPaymentDeclined
	default:
		return ErrInvalidCard
	}
}
