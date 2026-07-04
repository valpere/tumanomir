package payments

import (
	"database/sql"
	"time"
)

type UUID string

type Decimal string

type Currency string

type Provider string

const (
	ProviderStripe Provider = "stripe"
	ProviderPayPal Provider = "paypal"
	ProviderAdyen  Provider = "adyen"
)

type Status string

const (
	StatusOK       Status = "ok"
	StatusDeclined Status = "declined"
	StatusError    Status = "error"
)

type State string

const (
	StateReceived   State = "received"
	StateAuthorized State = "authorized"
	StateCaptured   State = "captured"
	StateDeclined   State = "declined"
)

type Transaction struct {
	ID             UUID
	Amount         Decimal
	Currency       Currency
	Provider       Provider
	IdempotencyKey UUID
}

type Receipt struct {
	ID            UUID
	TransactionID UUID
	Amount        Decimal
	Currency      Currency
	Status        Status
	State         State
	CreatedAt     time.Time
}

type PaymentLog struct {
	ID         UUID
	TxID       UUID
	Status     Status
	ErrorCode  sql.NullString
	CreatedAt  time.Time
}

type PaymentProcessor struct {
	DB *sql.DB
}

func (p *PaymentProcessor) AcceptTransaction(tx Transaction) (Receipt, error) {
	return Receipt{}, nil
}

func (p *PaymentProcessor) LogResult(txID UUID, status Status, errorCode *string) error {
	return nil
}
