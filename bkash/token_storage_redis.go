package bkash

import (
	"context"
	"fmt"
	"github.com/ShikhoTech/bd-payment-gateway/v2/bkash/models"
	"github.com/redis/go-redis/v9"
	"strconv"
	"time"
)

type redisTokenStorage struct {
	keyPrefix string
	rdb       *redis.Client
}

func (r *redisTokenStorage) Store(ctx context.Context, token models.Token) error {
	if err := r.rdb.HSet(ctx, r.keyPrefix,
		"token_type", token.TokenType,
		"expires_in", token.ExpiresIn,
		"id_token", token.IdToken,
		"refresh_token", token.RefreshToken,
		"expires_in_time", token.ExpiresInTime.Format(time.RFC3339),
		"created_at", token.CreatedAt.Format(time.RFC3339),
	).Err(); err != nil {
		return fmt.Errorf("could not store token in storage: %w", err)
	}

	if err := r.rdb.Expire(ctx, r.keyPrefix, time.Second*time.Duration(token.ExpiresIn-300)).Err(); err != nil {
		return fmt.Errorf("could not set token expiry: %w", err)
	}

	return nil
}

func (r *redisTokenStorage) Get(ctx context.Context) (token *models.Token, err error) {
	mapData, err := r.rdb.HGetAll(ctx, r.keyPrefix).Result()
	if err != nil {
		return nil, err
	}

	var (
		expiresIn                int64
		expiresInTime, createdAt time.Time
	)

	if value, ok := mapData["expires_in"]; ok {
		expiresIn, err = strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("could not parse token expires_in: %w, value: %v", err, value)
		}
	}

	if value, ok := mapData["expires_in_time"]; ok {
		expiresInTime, err = time.Parse(time.RFC3339, value)
		if err != nil {
			return nil, fmt.Errorf("could not parse token expires_in_time: %w, value: %v", err, value)
		}
	}

	if value, ok := mapData["created_at"]; ok {
		createdAt, err = time.Parse(time.RFC3339, value)
		if err != nil {
			return nil, fmt.Errorf("could not parse token created_at: %w, value: %v", err, value)
		}
	}

	token = &models.Token{
		TokenType:     mapData["token_type"],
		ExpiresIn:     int(expiresIn),
		IdToken:       mapData["id_token"],
		RefreshToken:  mapData["refresh_token"],
		ExpiresInTime: expiresInTime,
		CreatedAt:     createdAt,
	}
	return
}

func newRedisTokenStorage(rdb *redis.Client, keyPrefix string) TokenStorage {
	var prefix = "payment_service.bkash_token"
	if keyPrefix == "" {
		keyPrefix += ".local"
	} else {
		keyPrefix += "." + keyPrefix
	}
	return &redisTokenStorage{rdb: rdb, keyPrefix: prefix}
}
