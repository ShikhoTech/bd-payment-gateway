package bkash

import (
	"context"
	"github.com/ShikhoTech/bd-payment-gateway/v2/bkash/models"
)

type TokenStorage interface {
	Store(ctx context.Context, token models.Token) error
	Get(ctx context.Context) (token *models.Token, err error)
}
