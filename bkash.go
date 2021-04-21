package go_sslcom

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sh0umik/go-sslcom/models"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
)

const BKASH_SANDBOX_GATEWAY = "https://tokenized.sandbox.bka.sh"
const BKASH_LIVE_GATEWAY = "https://tokenized.sandbox.bka.sh"
const BKASH_GRANT_TOKEN_URI = "v1.2.0-beta/tokenized/checkout/token/grant"
const BKASH_REFRESH_TOKEN_URI = "v1.2.0-beta/tokenized/checkout/token/refresh"
const BKASH_CREATE_AGREEMENT_URI = "v1.2.0-beta/tokenized/checkout/create"
const BKASH_EXECUTE_AGREEMENT_URI = "v1.2.0-beta/tokenized/checkout/execute"
const BKASH_QUERY_AGREEMENT_URI = "v1.2.0-beta/tokenized/checkout/agreement/status"
const BKASH_CANCEL_AGREEMENT_URI = "v1.2.0-beta/tokenized/checkout/agreement/cancel"

var EMPTY_REQUIRED_FIELD = errors.New("empty required field")

type Bkash struct {
	Username  string
	Password  string
	AppKey    string
	AppSecret string
}

func GetBkash(username, password, appKey, appSecret string) *Bkash {
	return &Bkash{
		Username:  username,
		Password:  password,
		AppKey:    appKey,
		AppSecret: appSecret,
	}
}

func (b *Bkash) GrantToken(isLiveStore bool) (*models.Token, error) {
	// Mandatory field validation
	if b.AppKey == "" || b.AppSecret == "" || b.Username == "" || b.Password == "" {
		return nil, EMPTY_REQUIRED_FIELD
	}

	var data = make(map[string]string)

	data["app_key"] = b.AppKey
	data["app_secret"] = b.AppSecret

	var storeUrl string
	if isLiveStore {
		storeUrl = BKASH_LIVE_GATEWAY
	} else {
		storeUrl = BKASH_SANDBOX_GATEWAY
	}
	u, _ := url.ParseRequestURI(storeUrl)
	u.Path = BKASH_GRANT_TOKEN_URI

	grantTokenURL := u.String()

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	r, err := http.NewRequest("POST", grantTokenURL, bytes.NewReader(jsonData))
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

func (b *Bkash) RefreshToken(token *models.Token, isLiveStore bool) (*models.Token, error) {
	// Mandatory field validation
	if b.AppKey == "" || b.AppSecret == "" || token.RefreshToken == "" || b.Username == "" || b.Password == "" {
		return nil, EMPTY_REQUIRED_FIELD
	}

	var data = make(map[string]string)

	data["app_key"] = b.AppKey
	data["app_secret"] = b.AppSecret
	data["refresh_token"] = token.RefreshToken

	var storeUrl string
	if isLiveStore {
		storeUrl = BKASH_LIVE_GATEWAY
	} else {
		storeUrl = BKASH_SANDBOX_GATEWAY
	}
	u, _ := url.ParseRequestURI(storeUrl)
	u.Path = BKASH_REFRESH_TOKEN_URI

	refreshTokenURL := u.String()

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

func (b *Bkash) CreateAgreement(request *models.CreateAgreementRequest, token *models.Token, isLiveStore bool) (*models.CreateAgreementResponse, error) {
	// Mandatory field validation
	if b.AppKey == "" || token.IdToken == "" || request.Mode == "" || request.CallbackUrl == "" {
		return nil, EMPTY_REQUIRED_FIELD
	}

	// Mode validation
	if request.Mode != "0000" {
		return nil, errors.New("invalid mode value")
	}

	var storeUrl string
	if isLiveStore {
		storeUrl = BKASH_LIVE_GATEWAY
	} else {
		storeUrl = BKASH_SANDBOX_GATEWAY
	}
	u, _ := url.ParseRequestURI(storeUrl)
	u.Path = BKASH_CREATE_AGREEMENT_URI
	//u.RawQuery = data.Encode()

	createAgrrementURL := u.String()

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	r, err := http.NewRequest("POST", createAgrrementURL, bytes.NewReader(jsonData))
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

func (b *Bkash) ExecuteAgreement(request *models.ExecuteAgreementRequest, token *models.Token, isLiveStore bool) (*models.ExecuteAgreementResponse, error) {
	// Mandatory field validation
	if b.AppKey == "" || token.IdToken == "" || request.PaymentID == "" {
		return nil, EMPTY_REQUIRED_FIELD
	}

	var storeUrl string
	if isLiveStore {
		storeUrl = BKASH_LIVE_GATEWAY
	} else {
		storeUrl = BKASH_SANDBOX_GATEWAY
	}
	u, _ := url.ParseRequestURI(storeUrl)
	u.Path = BKASH_EXECUTE_AGREEMENT_URI
	//u.RawQuery = data.Encode()

	executeAgrrementURL := u.String()

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	r, err := http.NewRequest("POST", executeAgrrementURL, bytes.NewReader(jsonData))
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

func (b *Bkash) QueryAgreement(request *models.QueryAgreementRequest, token *models.Token, isLiveStore bool) (*models.QueryAgreementResponse, error) {
	// Mandatory field validation
	if b.AppKey == "" || token.IdToken == "" || request.AgreementID == "" {
		return nil, EMPTY_REQUIRED_FIELD
	}

	var storeUrl string
	if isLiveStore {
		storeUrl = BKASH_LIVE_GATEWAY
	} else {
		storeUrl = BKASH_SANDBOX_GATEWAY
	}
	u, _ := url.ParseRequestURI(storeUrl)
	u.Path = BKASH_QUERY_AGREEMENT_URI
	//u.RawQuery = data.Encode()

	queryAgreementURL := u.String()

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

func (b *Bkash) CancelAgreement(request *models.CancelAgreementRequest, token *models.Token, isLiveStore bool) (*models.CancelAgreementResponse, error) {
	// Mandatory field validation
	if b.AppKey == "" || token.IdToken == "" || request.AgreementID == "" {
		return nil, EMPTY_REQUIRED_FIELD
	}

	var storeUrl string
	if isLiveStore {
		storeUrl = BKASH_LIVE_GATEWAY
	} else {
		storeUrl = BKASH_SANDBOX_GATEWAY
	}
	u, _ := url.ParseRequestURI(storeUrl)
	u.Path = BKASH_CANCEL_AGREEMENT_URI
	//u.RawQuery = data.Encode()

	cancelAgrrementURL := u.String()

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	r, err := http.NewRequest("POST", cancelAgrrementURL, bytes.NewReader(jsonData))
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
