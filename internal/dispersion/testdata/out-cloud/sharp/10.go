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
	Amount         Decimal
	Currency       Currency
	Status         PaymentStatus
	IdempotencyKey UUID
	CreatedAt      time.Time
}

type PaymentStatus string

const (
	StatusReceived   PaymentStatus = "received"
	StatusAuthorized PaymentStatus = "authorized"
	StatusCaptured   PaymentStatus = "captured"
	StatusDeclined   PaymentStatus = "declined"
)

type LogStatus string

const (
	LogStatusOK       LogStatus = "ok"
	LogStatusDeclined LogStatus = "declined"
	LogStatusError    LogStatus = "error"
)

type PaymentLog struct {
	TxID       UUID
	Status     LogStatus
	ErrorCode  *string
	CreatedAt  time.Time
}

type PaymentProcessor interface {
	AcceptTransaction(tx Transaction) (Receipt, error)
	LogResult(txID UUID, status LogStatus, errorCode *string)
}
