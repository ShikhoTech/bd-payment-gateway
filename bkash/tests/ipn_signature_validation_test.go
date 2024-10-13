package tests

import (
	"encoding/json"
	"github.com/sh0umik/bd-payment-gateway/bkash"
	"github.com/sh0umik/bd-payment-gateway/bkash/models"
	"testing"
)

func TestMessageSignatureValidation(t *testing.T) {
	notificationJson := `
	{
	  "Type" : "Notification",
	  "MessageId" : "1eda938a-2b6d-5766-9d5e-cc0fd7eca13c",
	  "TopicArn" : "arn:aws:sns:ap-southeast-1:797962984373:bpt_01318693581",
	  "Message" : "{\"debitMSISDN\":\"01966734459\",\"creditOrganizationName\":\"Shikho Technologies Bangladesh Limited-RM48040\",\"creditShortCode\":\"01318693581\",\"dateTime\":\"20241013225009\",\"trxID\":\"BJD3FMWX7T\",\"transactionStatus\":\"Completed\",\"transactionType\":\"10003130\",\"payerType\":\"Customer\",\"currency\":\"BDT\",\"amount\":\"5000.0\",\"merchantInvoiceNumber\":\"InvZV33QX7T7Q\"}",
	  "Timestamp" : "2024-10-13T16:50:09.282Z",
	  "SignatureVersion" : "1",
	  "Signature" : "e/8PwIx7kXDbJFYSDyEphCnoN2O70aBbDHeTDtjUYy33BZF20LnquW/ceoZfTmxX9Mt32PM42BhueRBqKdma5VlFmdMcM7ZAWKnLC30CTObwfVi5SpXFRGT0xgeQOP0t5HOEpkgpo8t05yYdmhFj/eoUY4upHDpinFrvQ3MT9cCamZF82hrwlHYjL1KIJc7cxXxYAkI8Gxe0BZ7AStYJ0g2jKlukzgECKoncSCpnhaB6lVmrS+nPMvp4WdTa58lAXUzZKnJ1TI0OAfObO/cJ+H6FLCHxrKtiB1tZH9ZCHfQmVYOn65B+M7IV/8mPaLiURNynIwM4pjUBmfIYQxbVXQ==",
	  "SigningCertURL" : "https://sns.ap-southeast-1.amazonaws.com/SimpleNotificationService-60eadc530605d63b8e62a523676ef735.pem",
	  "UnsubscribeURL" : "https://sns.ap-southeast-1.amazonaws.com/?Action=Unsubscribe&SubscriptionArn=arn:aws:sns:ap-southeast-1:797962984373:bpt_01318693581:9ff11129-28b1-4589-9d5d-bb4761b6e4e2"
	}`
	var notificationPayload models.BkashIPNPayload

	err := json.Unmarshal([]byte(notificationJson), &notificationPayload)
	if err != nil {
		t.Fatal(err)
	}

	verifyErr := bkash.GetBkash("", "", "", "", false).IsMessageSignatureValid(&notificationPayload)
	if verifyErr != nil {
		t.Fatal(verifyErr)
	}
	t.Log("valid payload")
}
