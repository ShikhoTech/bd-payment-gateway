package models

type RefundTransactionResponse struct {
	// CompletedTime When the action of refund is done
	CompletedTime string `json:"completedTime,omitempty"`
	// TransactionStatus Last status of the transaction
	TransactionStatus string `json:"transactionStatus,omitempty"`
	// OriginalTrxID The main or originator transaction ID of that payment
	OriginalTrxID string `json:"originalTrxID,omitempty"`
	// RefundTrxID The transaction ID of that refund action itself
	RefundTrxID string `json:"refundTrxID,omitempty"`
	// Amount How much money refunded
	Amount string `json:"amount,omitempty"`
	// Currency The refund amount currency
	Currency string `json:"currency,omitempty"`
	// Charge If there are any charge to refund that amount, it will reflect here.
	Charge string `json:"charge,omitempty"`

	StatusCode    string `json:"statusCode,omitempty"`
	StatusMessage string `json:"statusMessage,omitempty"`
}

type RefundStatusResponse struct {
	RefundTransactionResponse
}
