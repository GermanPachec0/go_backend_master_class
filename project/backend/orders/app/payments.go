package app

import (
	"context"

	"github.com/shopspring/decimal"
)

type PaymentClient interface {
	CapturePayment(ctx context.Context, nonce string, amount decimal.Decimal, merchantID string) error
}
