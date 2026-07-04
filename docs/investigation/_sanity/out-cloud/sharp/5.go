package payments

import (
	"math/big"
	"time"
)

type UUID string

type Currency string

type Provider string

const (
	ProviderStripe Provider = "stripe"
	ProviderPayPal Provider = "paypal"
	ProviderAdyen  Provider = "adyen"
)

type Status string

const (
	StatusOk       Status = "ok"
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
	Amount         *big.Rat
	Currency       Currency
	Provider       Provider
	IdempotencyKey UUID
}

type Receipt struct {
	ID            UUID
	TransactionID UUID
	State         State
	CreatedAt     time.Time
}

type PaymentLog struct {
	TxID       UUID
	Status     Status
	ErrorCode  *string
	CreatedAt  time.Time
}

func AcceptTransaction(tx Transaction) (Receipt, error) {
}

func LogResult(txID UUID, status Status, errorCode *string) {
}
