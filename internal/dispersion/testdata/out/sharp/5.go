package payment

import (
	"time"
)

type Transaction struct {
	ID                string
	Amount            float64
	Currency          string
	Provider          string
	IdempotencyKey    string
}

type Receipt struct {
	ID        string
	Status    string
	ErrorCode *string
	CreatedAt time.Time
}

type PaymentLog struct {
	TxID        string
	Status      string
	ErrorCode   *string
	CreatedAt   time.Time
}

func AcceptTransaction(tx Transaction) (Receipt, error) {
	return Receipt{}, nil
}

func LogResult(txID, status, errorCode string) {
}

func GetReceiptByIdepotencyKey(key string) (Receipt, error) {
	return Receipt{}, nil
}
