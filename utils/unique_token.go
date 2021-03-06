package utils

import (
	"crypto/sha256"
	"fmt"
	"math/rand"

	"vobook/database"

	"github.com/go-pg/pg/v9"
)

const (
	UniqueTokenColumnDefault = "token"
	UniqueTokenLengthDefault = 64
)

type UniqueTokenOpts struct {
	Column string
	Length int
}

// UniqueToken creates unique token for given table.column
func UniqueToken(table string, opts ...UniqueTokenOpts) (string, error) {
	var opt UniqueTokenOpts
	if len(opts) == 1 {
		opt = opts[0]
	} else {
		opt = UniqueTokenOpts{
			Column: UniqueTokenColumnDefault,
			Length: UniqueTokenLengthDefault,
		}
	}

	if opt.Length < 1 {
		return "", fmt.Errorf("cannot make token with %d chars", opt.Length)
	}

	maxTries := len(Chars()) * opt.Length
	for i := 0; i < maxTries; i++ {
		token := randomString(opt.Length)
		_, err := database.ORM().ExecOne("SELECT * FROM ? WHERE ? = ?", pg.Ident(table), pg.Ident(opt.Column), token)
		if err == pg.ErrNoRows {
			return token, nil
		}

		if err != nil {
			return "", err
		}
	}

	return "", fmt.Errorf("unable to find unique token within %d attempts", maxTries)
}

func RandomToken(length int) string {
	chars := Chars()
	token := make([]byte, length)
	for i := 0; i < length; i++ {
		token[i] = chars[rand.Intn(len(chars))]
	}
	return string(token)
}

func RandomHash() string {
	token := RandomToken(64)
	hash := sha256.Sum256([]byte(token))

	return fmt.Sprintf("%x", hash)
}
