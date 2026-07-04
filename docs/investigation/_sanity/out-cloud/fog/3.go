package payments

import (
	"context"
)

type ProviderID string

type TransactionStatus string

const (
	TransactionStatusPending   TransactionStatus = "pending"
	TransactionStatusSuccess   TransactionStatus = "success"
	TransactionStatusFailed    TransactionStatus = "failed"
	TransactionStatusCancelled TransactionStatus = "cancelled"
)

type Money struct {
	Amount   int64
	Currency string
}

type Transaction struct {
	ID       string
	Provider ProviderID
	Amount   Money
	Status   TransactionStatus
	Metadata map[string]string
}

type Provider interface {
	ID() ProviderID
	Process(ctx context.Context, tx Transaction) (TransactionResult, error)
	Supports(tx Transaction) bool
}

type TransactionResult struct {
	TransactionID string
	Status        TransactionStatus
	ProviderRef   string
	ErrorCode     string
	ErrorMessage  string
}

type OperationLogEntry struct {
	Timestamp     int64
	TransactionID string
	Provider      ProviderID
	Status        TransactionStatus
	Result        TransactionResult
	Error         error
}

type OperationLogger interface {
	Log(entry OperationLogEntry) error
}

type Processor struct {
	providers []Provider
	logger    OperationLogger
}

func NewProcessor(logger OperationLogger) *Processor {
	return &Processor{}
}

func (p *Processor) RegisterProvider(provider Provider) {}

func (p *Processor) Process(ctx context.Context, tx Transaction) (TransactionResult, error) {
	return TransactionResult{}, nil
}

func (p *Processor) selectProvider(tx Transaction) (Provider, bool) {
	return nil, false
}
