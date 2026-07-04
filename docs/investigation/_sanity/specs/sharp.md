# Система обробки платежів користувачів

@schema Transaction {
  id: UUID,
  amount: Decimal(10,2) @constraint(min: 0.01),
  currency: String(3) @constraint(pattern: "^[A-Z]{3}$"),
  provider: Enum["stripe", "paypal", "adyen"],
  idempotency_key: UUID @constraint(unique)
}

1. [REQ-PAY-01] Система валідує вхідний запит за схемою Transaction.
   -> [FUN-PAY-01] AcceptTransaction(tx Transaction) (Receipt, error)
2. [REQ-PAY-02] Результати операцій записуються в таблицю payment_logs
   (tx_id UUID FK, status Enum["ok","declined","error"], error_code String?, created_at Timestamp).
   -> [FUN-PAY-02] LogResult(txID, status, errorCode)
3. [REQ-PAY-03] Повторний запит з тим самим idempotency_key повертає
   попередній Receipt без повторного списання.
   -> [FUN-PAY-03] state machine: received -> authorized -> captured | declined
