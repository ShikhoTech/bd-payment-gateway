package bkash

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/ShikhoTech/bd-payment-gateway/v2/bkash/models"
	"github.com/redis/go-redis/v9"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type tokenizer interface {
	GetToken() (*models.Token, error)
}

type redisTokenizer struct {
	Username  string
	Password  string
	AppKey    string
	AppSecret string

	isLiveStore bool

	redisClient *redis.Client
}

func NewRedisTokenizer(username, password, appKey, appSecret string, isLiveStore bool, redisClient *redis.Client) tokenizer {
	return &redisTokenizer{
		Username:    username,
		Password:    password,
		AppKey:      appKey,
		AppSecret:   appSecret,
		isLiveStore: isLiveStore,
		redisClient: redisClient,
	}
}

func (r *redisTokenizer) getTokenFromRedis() (*models.Token, error) {
	token := &models.Token{}
	err := r.redisClient.Get(context.Background(), "token_key").Scan(token)
	if err != nil {
		return nil, err
	}
	return token, nil
}

func (r *redisTokenizer) storeTokenInRedis(token *models.Token) error {
	err := r.redisClient.Set(context.Background(), "token_key", token, time.Duration(token.ExpiresIn)*time.Second).Err()
	if err != nil {
		return err
	}
	return nil
}

func (r *redisTokenizer) GetToken() (*models.Token, error) {
	token := &models.Token{}
	var err error

	// Check if token exists in redis
	token, _ = r.getTokenFromRedis()
	if token != nil {
		log.Print("Got token from redis")
	}

	// get  token from API if
	// 1. token is nil
	// 2. token is expired
	// 3. token is about to expire in 15 minutes and we generate a random number
	// between 0 and 100 and if the number is less than 10, we fetch a new token from the API

	// If token is not found in redis, fetch from bKash API
	if token == nil ||
		token.ExpiresAt.Before(time.Now().UTC()) ||
		(token.ExpiresAt.Before(time.Now().UTC().Add(15*time.Minute)) && rand.Intn(100) < 10) {
		token, err = r.getTokenFromAPIMock()
		if err != nil {
			return nil, err
		}

		log.Println("Got token from API")

		now := time.Now().UTC()
		expiresAt := now.Add(time.Duration(token.ExpiresIn-900) * time.Second)

		token.CreatedAt = now
		token.ExpiresAt = expiresAt

		// Store token in redis
		err = r.storeTokenInRedis(token)
		if err != nil {
			return nil, err
		}
	}

	return token, nil
}

func (r *redisTokenizer) getTokenFromAPI() (*models.Token, error) {
	// Mandatory field validation
	if r.AppKey == "" || r.AppSecret == "" || r.Username == "" || r.Password == "" {
		return nil, EMPTY_REQUIRED_FIELD
	}

	var data = make(map[string]string)

	data["app_key"] = r.AppKey
	data["app_secret"] = r.AppSecret

	var storeUrl string
	if r.isLiveStore {
		storeUrl = BKASH_LIVE_GATEWAY
	} else {
		storeUrl = BKASH_SANDBOX_GATEWAY
	}
	u, _ := url.ParseRequestURI(storeUrl)
	u.Path += BKASH_GRANT_TOKEN_URI

	grantTokenURL := u.String()

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	req, err := http.NewRequest("POST", grantTokenURL, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Content-Length", strconv.Itoa(len(jsonData)))
	req.Header.Add("username", r.Username)
	req.Header.Add("password", r.Password)

	response, err := client.Do(req)
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

func (r *redisTokenizer) getTokenFromAPIMock() (*models.Token, error) {
	return &models.Token{
		TokenType:     "Bearer",
		ExpiresIn:     100,
		IdToken:       "_id_token_2",
		RefreshToken:  "_refresh_token_",
		StatusCode:    "200",
		StatusMessage: "Success",
	}, nil
}
