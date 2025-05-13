package token

import (
	"testing"
	"time"

	"github.com/gentcod/nlp-to-sql/util"
	"github.com/google/uuid"
	"github.com/o1egl/paseto"
	"github.com/stretchr/testify/require"
)

func TestPasetoGenerator(t *testing.T) {
	username := util.RandomUser()
	duration := time.Minute

	maker, err := NewPasetoGenerator(util.RandomStr(32))
	require.NoError(t, err)

	issuedAt := time.Now()
	expiredAt := issuedAt.Add(duration)

	token, payload, err := maker.CreateToken(username, uuid.New(), duration)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.NotEmpty(t, payload)

	payload, err = maker.VerifyToken(token)
	require.NoError(t, err)
	require.NotEmpty(t, payload)

	require.NotZero(t, payload.ID)
	require.Equal(t, username, payload.Username)
	require.WithinDuration(t, issuedAt, payload.IssuedAt, time.Second)
	require.WithinDuration(t, expiredAt, payload.ExpiredAt, time.Second)
}

func TestExpiredPasetoToken(t *testing.T) {
	duration := time.Minute

	maker, err := NewPasetoGenerator(util.RandomStr(32))
	require.NoError(t, err)

	token, payload, err := maker.CreateToken(util.RandomUser(), uuid.New(), -duration)
	require.NoError(t, err)
	require.NotEmpty(t, payload)
	require.NotEmpty(t, token)

	payload, err = maker.VerifyToken(token)
	require.Error(t, err)
	require.EqualError(t, err, ErrExpiredToken.Error())
	require.Nil(t, payload)
}

func TestInvalidPasetoToken(t *testing.T) {
	//When secretKey length is invalid
	secret1 := util.RandomStr(20)

	maker1, err := NewPasetoGenerator(secret1)
	require.Error(t, err)
	require.Nil(t, maker1)

	//When none signature token type is used
	payloadInvalidSign, err := NewPayload(util.RandomUser(), uuid.New(), time.Minute)
	require.NoError(t, err)

	//Create Paseto Token with a different version
	pasetoToken, err := paseto.NewV1().Encrypt([]byte(util.RandomStr(32)), payloadInvalidSign, nil)
	require.NoError(t, err)

	makerInvalidSign, err := NewPasetoGenerator(util.RandomStr(32))
	require.NoError(t, err)

	payloadInvalidSign, err = makerInvalidSign.VerifyToken(pasetoToken)
	require.Error(t, err)
	require.EqualError(t, err, ErrInvalidToken.Error())
	require.Nil(t, payloadInvalidSign)
}
