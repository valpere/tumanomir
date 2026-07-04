package payment

import (
	"time"
)

type TransactionStatus string
type PaymentProvider string

const (
	Success TransactionStatus = "success"
	Failure TransactionStatus = "failure"
	Pending TransactionStatus = "pending"
)

const (
	Stripe    PaymentProvider = "stripe"
	PayPal    PaymentProvider = "paypal"
	BankTrans PaymentProvider = "bank_transfer"
)

type Transaction struct {
	ID            string
	Amount        float64
	Currency      string
	Provider      PaymentProvider
	Status        TransactionStatus
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Description   string
	UserID        string
	ReferenceID   string
	Error         *string
	Logs          []TransactionLog
}

type TransactionLog struct {
	Timestamp time.Time
	Message   string
	Level     string
}

type PaymentProcessor interface {
	ProcessTransaction(tx *Transaction) error
	GetTransactionStatus(txID string) (TransactionStatus, error)
}

type Logger interface {
	LogTransaction(tx *Transaction) error
	LogError(err error, tx *Transaction) error
}

func NewPaymentProcessor(provider PaymentProvider) PaymentProcessor {
	return &paymentProcessor{}
}

func NewLogger() Logger {
	return &logger{}
}

type paymentProcessor struct{}

func (p *paymentProcessor) ProcessTransaction(tx *Transaction) error {
	return nil
}

func (p *paymentProcessor) GetTransactionStatus(txID string) (TransactionStatus, error) {
	return Pending, nil
}

type logger struct{}

func (l *logger) LogTransaction(tx *Transaction) error {
	return nil
}

func (l *logger) LogError(err error, tx *Transaction) error {
	return nil
}
