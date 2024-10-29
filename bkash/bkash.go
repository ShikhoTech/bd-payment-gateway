package bkash

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/ShikhoTech/bd-payment-gateway/v2/bkash/models"
	goCache "github.com/patrickmn/go-cache"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"time"
)

const (
	bkashSandboxGateway      = "https://tokenized.sandbox.bka.sh/v1.2.0-beta"
	bkashLiveGateway         = "https://tokenized.pay.bka.sh/v1.2.0-beta"
	bkashGrantTokenUri       = "/tokenized/checkout/token/grant"
	bkashRefreshTokenUri     = "/tokenized/checkout/token/refresh"
	bkashCreateAgreementUri  = "/tokenized/checkout/create"
	bkashExecuteAgreementUri = "/tokenized/checkout/execute"
	bkashQueryAgreementUri   = "/tokenized/checkout/agreement/status"
	bkashCancelAgreementUri  = "/tokenized/checkout/agreement/cancel"
	bkashCreatePaymentUri    = "/tokenized/checkout/create"
	bkashExecutePaymentUri   = "/tokenized/checkout/execute"
	bkashQueryPaymentUri     = "/tokenized/checkout/payment/status"
	bkashRefundPaymentUri    = "/tokenized/checkout/payment/refund"
)

var emptyRequiredField = errors.New("empty required field")
var timeoutError = errors.New("api request timeout")

type bkash struct {
	Username  string
	Password  string
	AppKey    string
	AppSecret string

	isLiveStore bool
	cache       *goCache.Cache
	token       *models.Token
}

func NewTokenizedCheckoutService(username, password, appKey, appSecret string, isLiveStore bool) TokenizedCheckoutService {
	c := goCache.New(goCache.NoExpiration, -1)
	return &bkash{
		Username:  username,
		Password:  password,
		AppKey:    appKey,
		AppSecret: appSecret,

		isLiveStore: isLiveStore,
		cache:       c,
	}
}

func (b *bkash) getToken() (token models.Token, err error) {
	// Mandatory field validation
	if b.AppKey == "" || b.AppSecret == "" || b.Username == "" || b.Password == "" {
		return token, emptyRequiredField
	}

	var data = make(map[string]string)

	data["app_key"] = b.AppKey
	data["app_secret"] = b.AppSecret

	grantTokenURL := b.uri(bkashGrantTokenUri)

	jsonData, err := json.Marshal(data)
	if err != nil {
		return
	}

	client := &http.Client{}
	r, err := http.NewRequest("POST", grantTokenURL, bytes.NewReader(jsonData))
	if err != nil {
		return
	}

	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Content-Length", strconv.Itoa(len(jsonData)))
	r.Header.Add("username", b.Username)
	r.Header.Add("password", b.Password)

	response, err := client.Do(r)
	if err != nil {
		return
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return
	}

	err = json.Unmarshal(body, &token)
	if err != nil {
		return
	}

	if token.StatusCode != "0000" {
		return token, errors.New(fmt.Sprintf("token generation failed, status: %s, reason: %s", token.StatusCode, token.StatusMessage))
	}
	return
}

func (b *bkash) RefreshToken(token *models.Token) (*models.Token, error) {
	// Mandatory field validation
	if b.AppKey == "" || b.AppSecret == "" || token.RefreshToken == "" || b.Username == "" || b.Password == "" {
		return nil, emptyRequiredField
	}

	var data = make(map[string]string)

	data["app_key"] = b.AppKey
	data["app_secret"] = b.AppSecret
	data["refresh_token"] = token.RefreshToken

	refreshTokenURL := b.uri(bkashRefreshTokenUri)

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	r, err := http.NewRequest("POST", refreshTokenURL, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}

	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Content-Length", strconv.Itoa(len(jsonData)))
	r.Header.Add("username", b.Username)
	r.Header.Add("password", b.Password)

	response, err := client.Do(r)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var resp models.Token
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

func (b *bkash) CreateAgreement(request *models.CreateAgreementRequest) (*models.CreateAgreementResponse, error) {
	// Mandatory field validation
	if b.AppKey == "" || request.Mode == "" || request.CallbackUrl == "" {
		return nil, emptyRequiredField
	}

	token, err := b.getToken()
	if err != nil {
		return nil, err
	}

	// Mode validation
	if request.Mode != "0000" {
		return nil, errors.New("invalid mode value")
	}

	createAgreementURL := b.uri(bkashCreateAgreementUri)

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	r, err := http.NewRequest("POST", createAgreementURL, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}

	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Content-Length", strconv.Itoa(len(jsonData)))
	r.Header.Add("Authorization", fmt.Sprintf("%s %s", token.TokenType, token.IdToken))
	r.Header.Add("X-APP-Key", b.AppKey)

	response, err := client.Do(r)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var resp models.CreateAgreementResponse
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

func (b *bkash) CreateAgreementValidationListener(r *http.Request) (*models.CreateAgreementValidationResponse, error) {
	if r.Method != "POST" {
		return nil, errors.New("method not allowed")
	}

	var agreementTValidationResponse models.CreateAgreementValidationResponse

	err := json.NewDecoder(r.Body).Decode(&agreementTValidationResponse)
	if err != nil {
		return nil, err
	}

	return &agreementTValidationResponse, nil
}

func (b *bkash) ExecuteAgreement(request *models.ExecuteAgreementRequest) (*models.ExecuteAgreementResponse, error) {
	// Mandatory field validation
	if b.AppKey == "" || request.PaymentID == "" {
		return nil, emptyRequiredField
	}

	token, err := b.getToken()
	if err != nil {
		return nil, err
	}

	executeAgreementURL := b.uri(bkashExecuteAgreementUri)

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	r, err := http.NewRequest("POST", executeAgreementURL, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}

	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Content-Length", strconv.Itoa(len(jsonData)))
	r.Header.Add("Authorization", fmt.Sprintf("%s %s", token.TokenType, token.IdToken))
	r.Header.Add("X-APP-Key", b.AppKey)

	response, err := client.Do(r)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var resp models.ExecuteAgreementResponse
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

func (b *bkash) QueryAgreement(request *models.QueryAgreementRequest) (*models.QueryAgreementResponse, error) {
	// Mandatory field validation
	if b.AppKey == "" || request.AgreementID == "" {
		return nil, emptyRequiredField
	}

	token, err := b.getToken()
	if err != nil {
		return nil, err
	}

	queryAgreementURL := b.uri(bkashQueryAgreementUri)

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	r, err := http.NewRequest("POST", queryAgreementURL, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}

	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Content-Length", strconv.Itoa(len(jsonData)))
	r.Header.Add("Authorization", fmt.Sprintf("%s %s", token.TokenType, token.IdToken))
	r.Header.Add("X-APP-Key", b.AppKey)

	response, err := client.Do(r)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var resp models.QueryAgreementResponse
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

func (b *bkash) CancelAgreement(request *models.CancelAgreementRequest) (*models.CancelAgreementResponse, error) {
	// Mandatory field validation
	if b.AppKey == "" || request.AgreementID == "" {
		return nil, emptyRequiredField
	}

	token, err := b.getToken()
	if err != nil {
		return nil, err
	}

	cancelAgreementURL := b.uri(bkashCancelAgreementUri)

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	r, err := http.NewRequest("POST", cancelAgreementURL, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}

	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Content-Length", strconv.Itoa(len(jsonData)))
	r.Header.Add("Authorization", fmt.Sprintf("%s %s", token.TokenType, token.IdToken))
	r.Header.Add("X-APP-Key", b.AppKey)

	response, err := client.Do(r)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var resp models.CancelAgreementResponse
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

func (b *bkash) CreatePayment(request *models.CreatePaymentRequest) (*models.CreatePaymentResponse, error) {
	// Mandatory field validation
	if b.AppKey == "" || request.CallbackURL == "" {
		return nil, emptyRequiredField
	}

	token, err := b.getToken()
	if err != nil {
		return nil, err
	}

	if request.AgreementID != "" {
		request.Mode = "0001" // tokenized checkout mode
	} else {
		request.Mode = "0011" // direct payment mode
	}

	createPaymentURL := b.uri(bkashCreatePaymentUri)

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	r, err := http.NewRequest("POST", createPaymentURL, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}

	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Content-Length", strconv.Itoa(len(jsonData)))
	r.Header.Add("Authorization", fmt.Sprintf("%s %s", token.TokenType, token.IdToken))
	r.Header.Add("X-APP-Key", b.AppKey)

	response, err := client.Do(r)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var resp models.CreatePaymentResponse
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != "0000" {
		return nil, errors.New(fmt.Sprintf("payment initiation failed, status: %s, reason: %s", resp.StatusCode, resp.StatusMessage))
	}

	return &resp, nil
}

func (b *bkash) ExecutePayment(request *models.ExecutePaymentRequest) (*models.ExecutePaymentResponse, error) {
	// Mandatory field validation
	if b.AppKey == "" || request.PaymentID == "" {
		return nil, emptyRequiredField
	}

	token, err := b.getToken()
	if err != nil {
		return nil, err
	}

	executePayment := b.uri(bkashExecutePaymentUri)

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	r, err := http.NewRequest("POST", executePayment, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(r.Context(), time.Second*30)
	defer cancel()

	r = r.WithContext(ctx)

	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Content-Length", strconv.Itoa(len(jsonData)))
	r.Header.Add("Authorization", fmt.Sprintf("%s %s", token.TokenType, token.IdToken))
	r.Header.Add("X-APP-Key", b.AppKey)

	response, err := client.Do(r)
	if err != nil {
		// if error is timeout then call query payment
		// if complete return success payload (*models.ExecutePaymentResponse, nil)
		// if initiated - return something that should be handled by client (maybe return some kind of timeout error)
		if errors.Is(err, context.DeadlineExceeded) {
			queryResp, err := b.QueryPayment(&models.QueryPaymentRequest{PaymentID: request.PaymentID})
			if err != nil {
				return nil, err
			}

			if queryResp.StatusCode == "0000" && queryResp.TransactionStatus == "Completed" {
				return &models.ExecutePaymentResponse{
					PaymentID:             queryResp.PaymentID,
					PayerReference:        queryResp.PayerReference,
					PaymentExecuteTime:    queryResp.PaymentExecuteTime,
					TrxID:                 queryResp.TrxID,
					TransactionStatus:     queryResp.TransactionStatus,
					Amount:                queryResp.Amount,
					Currency:              queryResp.Currency,
					Intent:                queryResp.Intent,
					MerchantInvoiceNumber: queryResp.MerchantInvoiceNumber,
					StatusCode:            queryResp.StatusCode,
					StatusMessage:         queryResp.StatusMessage,
					//AgreementID:           "",
					//CustomerMsisdn:        "",
					//AgreementExecuteTime:  "",
					//AgreementStatus:       "",
				}, nil
			} else {
				return nil, timeoutError
			}
		} else {
			return nil, err
		}
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var resp models.ExecutePaymentResponse
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

func (b *bkash) QueryPayment(request *models.QueryPaymentRequest) (*models.QueryPaymentResponse, error) {
	// Mandatory field validation
	if b.AppKey == "" || request.PaymentID == "" {
		return nil, emptyRequiredField
	}

	token, err := b.getToken()
	if err != nil {
		return nil, err
	}

	queryPaymentURL := b.uri(bkashQueryPaymentUri)

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	r, err := http.NewRequest("POST", queryPaymentURL, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}

	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Content-Length", strconv.Itoa(len(jsonData)))
	r.Header.Add("Authorization", fmt.Sprintf("%s %s", token.TokenType, token.IdToken))
	r.Header.Add("X-APP-Key", b.AppKey)

	response, err := client.Do(r)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var resp models.QueryPaymentResponse
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

func (b *bkash) RefundTransaction(request *models.RefundTransactionRequest) (*models.RefundTransactionResponse, error) {
	// Mandatory field validation
	if b.AppKey == "" || request.PaymentID == "" {
		return nil, emptyRequiredField
	}

	token, err := b.getToken()
	if err != nil {
		return nil, err
	}

	executePayment := b.uri(bkashRefundPaymentUri)

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	r, err := http.NewRequest("POST", executePayment, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(r.Context(), time.Second*30)
	defer cancel()

	r = r.WithContext(ctx)

	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Content-Length", strconv.Itoa(len(jsonData)))
	r.Header.Add("Authorization", fmt.Sprintf("%s %s", token.TokenType, token.IdToken))
	r.Header.Add("X-APP-Key", b.AppKey)

	response, err := client.Do(r)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var resp models.RefundTransactionResponse
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != "0000" {
		return nil, errors.New(fmt.Sprintf("refund initiation failed, status: %s, reason: %s", resp.StatusCode, resp.StatusMessage))
	}

	return &resp, nil
}

// getMessageBytesToSign returns a byte array containing a signature usable for signature verification
func getMessageBytesToSign(msg *models.BkashIPNPayload) []byte {
	var builtSignature bytes.Buffer
	signableKeys := []string{"Message", "MessageId", "Subject", "SubscribeURL", "Timestamp", "Token", "TopicArn", "Type"}
	for _, key := range signableKeys {
		reflectedStruct := reflect.ValueOf(msg)
		field := reflect.Indirect(reflectedStruct).FieldByName(key)
		value := field.String()
		if field.IsValid() && value != "" {
			builtSignature.WriteString(key + "\n")
			builtSignature.WriteString(value + "\n")
		}
	}
	return builtSignature.Bytes()
}

// IsMessageSignatureValid validates bkash IPN message signature. Returns true, nil if ok,
// otherwise returns false, error
func (b *bkash) IsMessageSignatureValid(msg *models.BkashIPNPayload) error {
	var cert *x509.Certificate
	if iFace, found := b.cache.Get(msg.SigningCertURL); found {
		if crt, ok := iFace.(*x509.Certificate); ok {
			cert = crt
		}
	}

	if cert == nil {
		resp, err := http.Get(msg.SigningCertURL)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return errors.New("unable to get certificate err: " + resp.Status)
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		p, _ := pem.Decode(body)
		cert, err = x509.ParseCertificate(p.Bytes)
		if err != nil {
			return err
		}

		b.cache.SetDefault(msg.SigningCertURL, cert)
	}

	base64DecodedSignature, err := base64.StdEncoding.DecodeString(msg.Signature)
	if err != nil {
		return err
	}

	if err := cert.CheckSignature(x509.SHA1WithRSA, getMessageBytesToSign(msg), base64DecodedSignature); err != nil {
		return err
	}

	return nil
}

func (b *bkash) uri(path string) string {
	var storeUrl string
	if b.isLiveStore {
		storeUrl = bkashLiveGateway
	} else {
		storeUrl = bkashSandboxGateway
	}

	u, _ := url.ParseRequestURI(storeUrl)
	u.Path += path

	return u.String()
}
