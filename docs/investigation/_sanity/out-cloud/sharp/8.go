package payment

import "time"

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
	StatusOk      Status = "ok"
	StatusDeclined Status = "declined"
	StatusError   Status = "error"
)

type TransactionState string

const (
	StateReceived   TransactionState = "received"
	StateAuthorized TransactionState = "authorized"
	StateCaptured   TransactionState = "captured"
	StateDeclined   TransactionState = "declined"
)

type Transaction struct {
	ID             UUID
	Amount         Decimal
	Currency       Currency
	Provider       Provider
	IdempotencyKey UUID
}

type Receipt struct {
	TransactionID    UUID
	IdempotencyKey   UUID
	Amount           Decimal
	Currency         Currency
	Provider         Provider
	State            TransactionState
	AuthorizedAt     *time.Time
	CapturedAt       *time.Time
	DeclinedAt       *time.Time
	DeclineReason    string
}

type PaymentLog struct {
	TxID       UUID
	Status     Status
	ErrorCode  *string
	CreatedAt  time.Time
}

type PaymentProcessor struct{}

func (p *PaymentProcessor) AcceptTransaction(tx Transaction) (Receipt, error) {
	return Receipt{}, nil
}

func (p *PaymentProcessor) LogResult(txID UUID, status Status, errorCode *string) error {
	return nil
}

func (p *PaymentProcessor) ProcessStateMachine(tx Transaction) (Receipt, error) {
	return Receipt{}, nil
}
