package payments

import "time"

type UUID string

type Decimal string

type Currency string

type Provider string

type Status string

const (
	ProviderStripe Provider = "stripe"
	ProviderPayPal Provider = "paypal"
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
	Currency       string
	Provider       Provider
	IdempotencyKey UUID
}

type Receipt struct {
	TransactionID  UUID
	Amount         Decimal
	Currency       string
	Provider       Provider
	IdempotencyKey UUID
	Status         Status
	CreatedAt      time.Time
}

type PaymentLog struct {
	TxID       UUID
	Status     Status
	ErrorCode  *string
	CreatedAt  time.Time
}

type PaymentState string

const (
	StateReceived   PaymentState = "received"
	StateAuthorized PaymentState = "authorized"
	StateCaptured   PaymentState = "captured"
	StateDeclined   PaymentState = "declined"
)

type PaymentStateMachine struct {
	State PaymentState
}

func AcceptTransaction(tx Transaction) (Receipt, error) {
	return Receipt{}, nil
}

func LogResult(txID UUID, status Status, errorCode *string) {}

func (sm *PaymentStateMachine) Authorize() error {
	return nil
}

func (sm *PaymentStateMachine) Capture() error {
	return nil
}

func (sm *PaymentStateMachine) Decline() error {
	return nil
}
