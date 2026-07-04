package payments

import (
	"context"
)

type TransactionStatus string

const (
	TransactionStatusSuccess TransactionStatus = "success"
	TransactionStatusFailed  TransactionStatus = "failed"
)

type Transaction struct {
	ID       string
	Provider string
	Amount   float64
	Currency string
}

type TransactionResult struct {
	TransactionID string
	Status        TransactionStatus
	Error         error
}

type Provider interface {
	Name() string
	Process(ctx context.Context, tx Transaction) TransactionResult
}

type Logger interface {
	LogSuccess(result TransactionResult)
	LogFailure(result TransactionResult)
}

type PaymentProcessor struct {
	providers []Provider
	logger    Logger
}

func NewPaymentProcessor(logger Logger) *PaymentProcessor {
	return &PaymentProcessor{}
}

func (p *PaymentProcessor) RegisterProvider(provider Provider) {}

func (p *PaymentProcessor) ProcessTransaction(ctx context.Context, tx Transaction) TransactionResult {
	return TransactionResult{}
}
