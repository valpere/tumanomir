package payment

import (
	"time"
)

type UUID = string

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
	TransactionID UUID
	Status        PaymentStatus
	Amount        Decimal
	Currency      Currency
	Provider      Provider
	CreatedAt     time.Time
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
	TxID      UUID
	Status    PaymentStatus
	ErrorCode *string
	CreatedAt time.Time
}

type PaymentStore interface {
	FindByIdempotencyKey(key UUID) (*Receipt, bool, error)
	SaveReceipt(receipt Receipt) error
	LogResult(log PaymentLog) error
	UpdateState(txID UUID, state PaymentState) error
}

func AcceptTransaction(tx Transaction) (Receipt, error) {
	return Receipt{}, nil
}

func LogResult(txID UUID, status PaymentStatus, errorCode *string) error {
	return nil
}

func TransitionState(current PaymentState, event PaymentEvent) (PaymentState, error) {
	return "", nil
}

type PaymentEvent string

const (
	EventAuthorize PaymentEvent = "authorize"
	EventCapture   PaymentEvent = "capture"
	EventDecline   PaymentEvent = "decline"
)
