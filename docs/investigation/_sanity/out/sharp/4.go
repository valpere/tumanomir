package payment

import (
	"time"
)

type UUID string
type Decimal string
type Enum string

type Transaction struct {
	ID                UUID      `json:"id"`
	Amount            Decimal   `json:"amount"`
	Currency          string    `json:"currency"`
	Provider          Enum      `json:"provider"`
	IdempotencyKey    UUID      `json:"idempotency_key"`
}

type Receipt struct {
	ID        UUID    `json:"id"`
	Status    Enum    `json:"status"`
	Error     *string `json:"error"`
	CreatedAt time.Time `json:"created_at"`
}

type PaymentLog struct {
	TxID        UUID    `json:"tx_id"`
	Status      Enum    `json:"status"`
	ErrorCode   *string `json:"error_code"`
	CreatedAt   time.Time `json:"created_at"`
}

const (
	ProviderStripe Enum = "stripe"
	ProviderPayPal Enum = "paypal"
	ProviderAdyen  Enum = "adyen"
)

const (
	StatusOK       Enum = "ok"
	StatusDeclined Enum = "declined"
	StatusError    Enum = "error"
)

func AcceptTransaction(tx Transaction) (Receipt, error) {
	panic("not implemented")
}

func LogResult(txID UUID, status Enum, errorCode *string) {
	panic("not implemented")
}
