package bkash

import (
	"github.com/sh0umik/bd-payment-gateway/bkash/models"
	"net/http"
)

type BkashTokenizedCheckoutService interface {
	// GrantToken creates a access token using bkash credentials
	GrantToken() (*models.Token, error)

	// RefreshToken refreshes the access token
	RefreshToken(token *models.Token) (*models.Token, error)

	// CreateAgreement Initiates an agreement request for a customer.
	CreateAgreement(request *models.CreateAgreementRequest, token *models.Token) (*models.CreateAgreementResponse, error)

	// CreateAgreementValidationListener is a handler func that receives paymentID & status
	// as a json post request and returns CreateAgreementValidationResponse object
	//
	// Deprecated: CreateAgreementValidationListener id deprecated, and should not be used.
	// Future release will drop the func.
	CreateAgreementValidationListener(r *http.Request) (*models.CreateAgreementValidationResponse, error)

	// ExecuteAgreement executes the agreement using the paymentID received from CreateAgreementValidationResponse
	ExecuteAgreement(request *models.ExecuteAgreementRequest, token *models.Token) (*models.ExecuteAgreementResponse, error)

	// QueryAgreement query agreement by agreementID
	QueryAgreement(request *models.QueryAgreementRequest, token *models.Token) (*models.QueryAgreementResponse, error)

	// CancelAgreement cancels an agreement by agreementID
	CancelAgreement(request *models.CancelAgreementRequest, token *models.Token) (*models.CancelAgreementResponse, error)

	// CreatePayment Initiates a payment request for a customer.
	// Mode value should be "0001".
	CreatePayment(request *models.CreatePaymentRequest, token *models.Token) (*models.CreatePaymentResponse, error)

	// ExecutePayment executes the agreement using the paymentID received from CreateAgreementValidationResponse
	ExecutePayment(request *models.ExecutePaymentRequest, token *models.Token) (*models.ExecutePaymentResponse, error)

	// QueryPayment query payment by paymentID
	QueryPayment(request *models.QueryPaymentRequest, token *models.Token) (*models.QueryPaymentResponse, error)

	IsMessageSignatureValid(msg *models.BkashIPNPayload) error
}
