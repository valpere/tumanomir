package payment

import (
	"time"
)

type UUID string

type Decimal string

type Currency string

type Provider string

const (
	ProviderStripe Provider = "stripe"
	ProviderPaypal Provider = "paypal"
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

type StateMachine struct {
	Current PaymentStatus
}

func AcceptTransaction(tx Transaction) (Receipt, error) {
	return Receipt{}, nil
}

func LogResult(txID UUID, status LogStatus, errorCode *string) error {
	return nil
}

func (sm *StateMachine) Transition(target PaymentStatus) error {
	return nil
}

func FindReceiptByIdempotencyKey(key UUID) (Receipt, bool) {
	return Receipt{}, false
}
