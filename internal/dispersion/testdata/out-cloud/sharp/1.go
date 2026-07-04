package payments

import (
	"database/sql"
	"time"
)

type UUID string
type Decimal string
type CurrencyCode string
type Provider string
type Status string
type ErrorCode string
type TransactionID string
type IdempotencyKey UUID

const (
	ProviderStripe Provider = "stripe"
	ProviderPaypal Provider = "paypal"
	ProviderAdyen  Provider = "adyen"
)

const (
	StatusOk       Status = "ok"
	StatusDeclined Status = "declined"
	StatusError    Status = "error"
)

type Transaction struct {
	ID             UUID
	Amount         Decimal
	Currency       CurrencyCode
	Provider       Provider
	IdempotencyKey IdempotencyKey
}

type Receipt struct {
	TransactionID UUID
	Amount        Decimal
	Currency      CurrencyCode
	Provider      Provider
	Status        Status
	CreatedAt     time.Time
}

type PaymentLog struct {
	TxID      UUID
	Status    Status
	ErrorCode sql.NullString
	CreatedAt time.Time
}

type PaymentState string

const (
	StateReceived   PaymentState = "received"
	StateAuthorized PaymentState = "authorized"
	StateCaptured   PaymentState = "captured"
	StateDeclined   PaymentState = "declined"
)

type PaymentProcessor interface {
	AcceptTransaction(tx Transaction) (Receipt, error)
	LogResult(txID UUID, status Status, errorCode sql.NullString) error
}

type PaymentStore interface {
	FindReceiptByIdempotencyKey(key IdempotencyKey) (Receipt, bool, error)
	SaveReceipt(receipt Receipt) error
	SaveLog(log PaymentLog) error
	GetState(txID UUID) (PaymentState, error)
	SetState(txID UUID, state PaymentState) error
}

func AcceptTransaction(tx Transaction) (Receipt, error) {
	return Receipt{}, nil
}

func LogResult(txID UUID, status Status, errorCode sql.NullString) error {
	return nil
}

func TransitionState(current PaymentState, event string) (PaymentState, error) {
	return "", nil
}
