package payment

import (
	"database/sql/driver"
	"time"
)

type UUID string
type Decimal string

func (d *Decimal) Scan(value interface{}) error { return nil }
func (d Decimal) Value() (driver.Value, error) { return nil, nil }

type Currency string
type Provider string
type Status string

const (
	ProviderStripe Provider = "stripe"
	ProviderPaypal Provider = "paypal"
	ProviderAdyen  Provider = "adyen"
)

const (
	StatusOK       Status = "ok"
	StatusDeclined Status = "declined"
	StatusError    Status = "error"
)

type Transaction struct {
	ID                UUID      `json:"id"`
	Amount            Decimal   `json:"amount"`
	Currency          Currency  `json:"currency"`
	Provider          Provider  `json:"provider"`
	IdempotencyKey    UUID      `json:"idempotency_key"`
}

type Receipt struct {
	TransactionID UUID      `json:"transaction_id"`
	Status        Status    `json:"status"`
	ErrorCode     *string   `json:"error_code,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

type PaymentLog struct {
	TxID      UUID      `json:"tx_id"`
	Status    Status    `json:"status"`
	ErrorCode *string   `json:"error_code,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func AcceptTransaction(tx Transaction) (Receipt, error) { return Receipt{}, nil }
func LogResult(txID, status string, errorCode *string) error { return nil }
