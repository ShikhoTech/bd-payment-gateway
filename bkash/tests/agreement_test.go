package tests

import (
	"github.com/sh0umik/bd-payment-gateway/bkash"
	"github.com/sh0umik/bd-payment-gateway/bkash/models"
	"os"
	"testing"
)

func TestAgreement(t *testing.T) {
	username := os.Getenv("BKASH_USERNAME")
	password := os.Getenv("BKASH_PASSWORD")
	appKey := os.Getenv("BKASH_APP_KEY")
	appSecret := os.Getenv("BKASH_APP_SECRET")

	bkashService := bkash.GetBkash(username, password, appKey, appSecret, false)
	paymentService := bkash.BkashTokenizedCheckoutService(bkashService)

	token, err := paymentService.GrantToken()
	if err != nil {
		t.Fatal(err)
	}

	if token == nil || len(token.IdToken) == 0 || len(token.RefreshToken) == 0 || token.StatusCode != "0000" {
		t.Fatal(err)
	}

	var createAgreementResponse *models.CreateAgreementResponse
	t.Run("test CreateAgreement", func(t *testing.T) {
		req := &models.CreateAgreementRequest{
			Mode:           "0000",
			PayerReference: "dsfsodjf-w3y2sdjf83493-sdhfis",
			CallbackUrl:    "https://api.shikho.net/payment",
			Currency:       "BDT",
			Intent:         "Shikho Subscription",
		}
		resp, err := paymentService.CreateAgreement(req, token)

		if err != nil {
			t.Error(err.Error())
			t.Fail()
		}

		if resp == nil || resp.StatusCode != "0000" {
			t.Fatal("Invalid create agreement response")
		}

		createAgreementResponse = resp
	})

	var executeAgreementReponse *models.ExecuteAgreementResponse
	t.Run("test ExecuteAgreement", func(t *testing.T) {
		req := &models.ExecuteAgreementRequest{
			PaymentID: createAgreementResponse.PaymentID,
		}
		resp, err := paymentService.ExecuteAgreement(req, token)

		if err != nil {
			t.Error(err.Error())
			t.Fail()
		}

		if resp == nil || resp.StatusCode != "0000" {
			t.Fatal("Invalid execute agreement response")
		}

		executeAgreementReponse = resp
	})

	var createPaymentResponse *models.CreatePaymentResponse
	t.Run("test CreatePayment", func(t *testing.T) {
		req := &models.CreatePaymentRequest{
			Mode:                    "0001",
			PayerReference:          "01723888888",
			CallbackURL:             "https://shikho.tech/payment",
			AgreementID:             executeAgreementReponse.AgreementID,
			Amount:                  "12",
			Currency:                "BDT",
			Intent:                  "sale",
			MerchantInvoiceNumber:   "Inv0124",
			MerchantAssociationInfo: "MI05MID54RF09123456One",
		}
		resp, err := paymentService.CreatePayment(req, token)

		if err != nil {
			t.Error(err.Error())
			t.Fail()
		}

		if resp == nil || resp.StatusCode != "0000" {
			t.Fatal("payment creattion failed")
		}

		createPaymentResponse = resp
	})

	t.Run("test executePayment", func(t *testing.T) {
		req := &models.ExecutePaymentRequest{
			PaymentID: createPaymentResponse.PaymentID,
		}
		resp, err := paymentService.ExecutePayment(req, token)

		if err != nil {
			t.Error(err.Error())
			t.Fail()
		}

		if resp == nil || resp.StatusCode != "0000" {
			t.Fatal("payment creattion failed")
		}
	})
}
