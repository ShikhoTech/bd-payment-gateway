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
	"github.com/ShikhoTech/bd-payment-gateway/bkash/models"
	goCache "github.com/patrickmn/go-cache"
	"github.com/redis/go-redis/v9"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"time"
)

const BKASH_SANDBOX_GATEWAY = "https://tokenized.sandbox.bka.sh/v1.2.0-beta"
const BKASH_LIVE_GATEWAY = "https://tokenized.pay.bka.sh/v1.2.0-beta"
const BKASH_GRANT_TOKEN_URI = "/tokenized/checkout/token/grant"
const BKASH_REFRESH_TOKEN_URI = "/tokenized/checkout/token/refresh"
const BKASH_CREATE_AGREEMENT_URI = "/tokenized/checkout/create"
const BKASH_EXECUTE_AGREEMENT_URI = "/tokenized/checkout/execute"
const BKASH_QUERY_AGREEMENT_URI = "/tokenized/checkout/agreement/status"
const BKASH_CANCEL_AGREEMENT_URI = "/tokenized/checkout/agreement/cancel"
const BKASH_CREATE_PAYMENT_URI = "/tokenized/checkout/create"
const BKASH_EXECUTE_PAYMENT_URI = "/tokenized/checkout/execute"
const BKASH_QUERY_PAYMENT_URI = "/tokenized/checkout/payment/status"

var EMPTY_REQUIRED_FIELD = errors.New("empty required field")
var TIMEOUT_ERROR = errors.New("api request timeout")

type Config struct {
	Username  string
	Password  string
	AppKey    string
	AppSecret string

	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDbNumber int

	IsLiveStore bool
}

func (c Config) Validate() error {
	if c.Username == "" {
		return fmt.Errorf("invalid Config: Username is required")
	}
	if c.Password == "" {
		return fmt.Errorf("invalid Config: Password is required")
	}
	if c.AppKey == "" {
		return fmt.Errorf("invalid Config: AppKey is required")
	}
	if c.AppSecret == "" {
		return fmt.Errorf("invalid Config: AppSecret is required")
	}
	if c.RedisHost == "" {
		return fmt.Errorf("invalid Config: RedisHost is required")
	}
	if c.RedisPort == "" {
		return fmt.Errorf("invalid Config: RedisPort is required")
	}
	return nil
}

type Bkash struct {
	config Config

	cache     *goCache.Cache
	tokenizer tokenizer

	storeUrl string
}

func GetBkash(config Config) (BkashTokenizedCheckoutService, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	// Setup in memory cache
	c := goCache.New(goCache.NoExpiration, -1)

	// setup tokenizer
	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", config.RedisHost, config.RedisPort),
		Password: config.RedisPassword,
		DB:       config.RedisDbNumber,
	})
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("could not connect to redis: %v", err)
	}

	t := NewRedisTokenizer(config.Username, config.Password, config.AppKey, config.AppSecret, config.IsLiveStore, redisClient)

	// set store url
	var storeUrl string
	if config.IsLiveStore {
		storeUrl = BKASH_LIVE_GATEWAY
	} else {
		storeUrl = BKASH_SANDBOX_GATEWAY
	}

	return &Bkash{config: config, cache: c, tokenizer: t, storeUrl: storeUrl}, nil
}

func (b *Bkash) CreateAgreement(request *models.CreateAgreementRequest) (*models.CreateAgreementResponse, error) {
	// Mandatory field validation
	if request.Mode == "" || request.CallbackUrl == "" {
		return nil, EMPTY_REQUIRED_FIELD
	}

	// Mode validation
	if request.Mode != "0000" {
		return nil, errors.New("invalid mode value")
	}

	u, _ := url.ParseRequestURI(b.storeUrl)
	u.Path += BKASH_CREATE_AGREEMENT_URI

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}

	r, err := b.newAuthorizedHttpPostRequest(u, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}

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

func (b *Bkash) CreateAgreementValidationListener(r *http.Request) (*models.CreateAgreementValidationResponse, error) {
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

func (b *Bkash) ExecuteAgreement(request *models.ExecuteAgreementRequest, ) (*models.ExecuteAgreementResponse, error) {
	// Mandatory field validation
	if request.PaymentID == "" {
		return nil, EMPTY_REQUIRED_FIELD
	}

	u, _ := url.ParseRequestURI(b.storeUrl)
	u.Path += BKASH_EXECUTE_AGREEMENT_URI

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	r, err := b.newAuthorizedHttpPostRequest(u, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}

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

func (b *Bkash) QueryAgreement(request *models.QueryAgreementRequest, ) (*models.QueryAgreementResponse, error) {
	// Mandatory field validation
	if request.AgreementID == "" {
		return nil, EMPTY_REQUIRED_FIELD
	}

	u, _ := url.ParseRequestURI(b.storeUrl)
	u.Path += BKASH_QUERY_AGREEMENT_URI

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}

	r, err := b.newAuthorizedHttpPostRequest(u, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}

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

func (b *Bkash) CancelAgreement(request *models.CancelAgreementRequest) (*models.CancelAgreementResponse, error) {
	// Mandatory field validation
	if request.AgreementID == "" {
		return nil, EMPTY_REQUIRED_FIELD
	}

	u, _ := url.ParseRequestURI(b.storeUrl)
	u.Path += BKASH_CANCEL_AGREEMENT_URI

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}

	r, err := b.newAuthorizedHttpPostRequest(u, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}

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

func (b *Bkash) CreatePayment(request *models.CreatePaymentRequest) (*models.CreatePaymentResponse, error) {
	// Mandatory field validation
	if request.Mode == "" || request.CallbackURL == "" {
		return nil, EMPTY_REQUIRED_FIELD
	}

	// Mode validation
	if request.Mode != "0001" && request.Mode != "0011" {
		return nil, errors.New("invalid mode value")
	}

	u, _ := url.ParseRequestURI(b.storeUrl)
	u.Path += BKASH_CREATE_PAYMENT_URI

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}

	r, err := b.newAuthorizedHttpPostRequest(u, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}

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

	return &resp, nil
}

func (b *Bkash) ExecutePayment(request *models.ExecutePaymentRequest) (*models.ExecutePaymentResponse, error) {
	// Mandatory field validation
	if request.PaymentID == "" {
		return nil, EMPTY_REQUIRED_FIELD
	}

	u, _ := url.ParseRequestURI(b.storeUrl)
	u.Path += BKASH_EXECUTE_PAYMENT_URI

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}

	r, err := b.newAuthorizedHttpPostRequest(u, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(r.Context(), time.Second*30)
	defer cancel()

	r = r.WithContext(ctx)

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
					// AgreementID:           "",
					// CustomerMsisdn:        "",
					// AgreementExecuteTime:  "",
					// AgreementStatus:       "",
				}, nil
			} else {
				return nil, TIMEOUT_ERROR
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

func (b *Bkash) QueryPayment(request *models.QueryPaymentRequest) (*models.QueryPaymentResponse, error) {
	// Mandatory field validation
	if request.PaymentID == "" {
		return nil, EMPTY_REQUIRED_FIELD
	}

	u, _ := url.ParseRequestURI(b.storeUrl)
	u.Path += BKASH_QUERY_PAYMENT_URI

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	r, err := b.newAuthorizedHttpPostRequest(u, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}

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
func (b *Bkash) IsMessageSignatureValid(msg *models.BkashIPNPayload) error {
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

func (b *Bkash) newAuthorizedHttpPostRequest(url *url.URL, body io.Reader) (*http.Request, error) {
	r, err := http.NewRequest("POST", url.String(), body)
	if err != nil {
		return nil, err
	}

	// get the token
	token, err := b.tokenizer.GetToken()
	if err != nil {
		return nil, err
	}

	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Authorization", fmt.Sprintf("%s %s", token.TokenType, token.IdToken))
	r.Header.Add("X-APP-Key", b.config.AppKey)

	return r, nil
}
