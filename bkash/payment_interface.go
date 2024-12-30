package bkash

import (
	"github.com/ShikhoTech/bd-payment-gateway/v2/bkash/models"
	"net/http"
)

const (
	StatusSuccessful = "Successful"
	StatusFailure    = "failure"
	StatusCancel     = "cancel"
)

type BkashTokenizedCheckoutService interface {
	// CreateAgreement Initiates an agreement request for a customer.
	CreateAgreement(request *models.CreateAgreementRequest) (*models.CreateAgreementResponse, error)

	// CreateAgreementValidationListener is a handler func that receives paymentID & status
	// as a json post request and returns CreateAgreementValidationResponse object
	//
	// Deprecated: CreateAgreementValidationListener id deprecated, and should not be used.
	// Future release will drop the func.
	CreateAgreementValidationListener(r *http.Request) (*models.CreateAgreementValidationResponse, error)

	// ExecuteAgreement executes the agreement using the paymentID received from CreateAgreementValidationResponse
	ExecuteAgreement(request *models.ExecuteAgreementRequest) (*models.ExecuteAgreementResponse, error)

	// QueryAgreement query agreement by agreementID
	QueryAgreement(request *models.QueryAgreementRequest) (*models.QueryAgreementResponse, error)

	// CancelAgreement cancels an agreement by agreementID
	CancelAgreement(request *models.CancelAgreementRequest) (*models.CancelAgreementResponse, error)

	// CreatePayment Initiates a payment request for a customer.
	// Mode value should be "0001".
	CreatePayment(request *models.CreatePaymentRequest) (*models.CreatePaymentResponse, error)

	// ExecutePayment executes the agreement using the paymentID received from CreateAgreementValidationResponse
	ExecutePayment(request *models.ExecutePaymentRequest) (*models.ExecutePaymentResponse, error)

	// QueryPayment query payment by paymentID
	QueryPayment(request *models.QueryPaymentRequest) (*models.QueryPaymentResponse, error)

	IsMessageSignatureValid(msg *models.BkashIPNPayload) error
}
