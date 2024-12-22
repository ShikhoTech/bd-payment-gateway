package bkash

import (
	"fmt"
	"github.com/redis/go-redis/v9"
	"testing"
)

func TestRedisTokenizer_GetToken(t *testing.T) {
	rClient := redis.NewClient(&redis.Options{Addr: "localhost:6379", Password: "", DB: 0})

	tokenizer := NewRedisTokenizer(
		"uname",
		"pass",
		"",
		"",
		false,
		rClient,
	)

	t.Run("Get from API", func(t *testing.T) {
		got, err := tokenizer.GetToken()
		if err != nil {
			t.Errorf("GetToken() error = %v", err)
			t.Fail()
		}

		fmt.Println(got)
	})
}
