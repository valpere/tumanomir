package payments

import (
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

type Transaction struct {
	ID             UUID
	Amount         Decimal
	Currency       Currency
	Provider       Provider
	IdempotencyKey UUID
}

type Receipt struct {
	TransactionID  UUID
	IdempotencyKey UUID
	Amount         Decimal
	Currency       Currency
	Status         PaymentStatus
	CreatedAt      time.Time
}

type PaymentStatus string

const (
	StatusOK       PaymentStatus = "ok"
	StatusDeclined PaymentStatus = "declined"
	StatusError    PaymentStatus = "error"
)

type PaymentState string

const (
	StateReceived   PaymentState = "received"
	StateAuthorized PaymentState = "authorized"
	StateCaptured   PaymentState = "captured"
	StateDeclined   PaymentState = "declined"
)

type PaymentLog struct {
	TxID       UUID
	Status     PaymentStatus
	ErrorCode  *string
	CreatedAt  time.Time
}

func AcceptTransaction(tx Transaction) (Receipt, error) {
	return Receipt{}, nil
}

func LogResult(txID UUID, status PaymentStatus, errorCode *string) error {
	return nil
}
