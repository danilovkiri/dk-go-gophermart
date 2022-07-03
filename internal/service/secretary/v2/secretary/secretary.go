// Package secretary provides methods for ciphering.
package secretary

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/danilovkiri/dk-go-gophermart/internal/config"
	"github.com/danilovkiri/dk-go-gophermart/internal/models/modelclaims"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"net/http"
	"time"
)

// Secretary defines object structure and its attributes.
type Secretary struct {
	aesgcm cipher.AEAD
	nonce  []byte
	key    []byte
}

// NewSecretaryService initializes a secretary service with ciphering functionality.
func NewSecretaryService(c *config.SecretConfig) (*Secretary, error) {
	key := sha256.Sum256([]byte(c.SecretKey))
	aesblock, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	aesgcm, err := cipher.NewGCM(aesblock)
	if err != nil {
		return nil, err
	}
	nonce := key[len(key)-aesgcm.NonceSize():]
	return &Secretary{
		aesgcm: aesgcm,
		nonce:  nonce,
		key:    []byte(c.SecretKey),
	}, nil
}

// Encode ciphers data using the previously established cipher.
func (s *Secretary) Encode(data string) string {
	encoded := s.aesgcm.Seal(nil, s.nonce, []byte(data), nil)
	return hex.EncodeToString(encoded)
}

// Decode deciphers data using the previously established cipher.
func (s *Secretary) Decode(msg string) (string, error) {
	msgBytes, err := hex.DecodeString(msg)
	if err != nil {
		return "", err
	}
	decoded, err := s.aesgcm.Open(nil, s.nonce, msgBytes, nil)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

// NewCookie generates a new userID and a corresponding encoded cookie.
func (s *Secretary) NewCookie() (*http.Cookie, string) {
	userID := uuid.New().String()
	token := s.Encode(userID)
	newCookie := &http.Cookie{
		Name:    "userID",
		Value:   token,
		Path:    "/",
		Expires: time.Now().Add(30 * time.Minute),
		//Expires: time.Now().Add(30 * time.Second),
	}
	return newCookie, userID
}

// GetCookieForUser generates an encoded cookie for a userID.
func (s *Secretary) GetCookieForUser(userID string) *http.Cookie {
	token := s.Encode(userID)
	userCookie := &http.Cookie{
		Name:    "userID",
		Value:   token,
		Path:    "/",
		Expires: time.Now().Add(30 * time.Minute),
	}
	return userCookie
}

func (s *Secretary) ValidateToken(accessToken string) (string, error) {
	token, err := jwt.ParseWithClaims(accessToken, &modelclaims.MyCustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.key, nil
	})
	if err != nil {
		return "", err
	}
	if claims, ok := token.Claims.(*modelclaims.MyCustomClaims); ok && token.Valid {
		return claims.UserID, nil
	}
	return "", errors.New("invalid access token")
}

func (s *Secretary) NewToken() (string, string, error) {
	userID := uuid.New().String()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &modelclaims.MyCustomClaims{
		UserID: userID,
		StandardClaims: jwt.StandardClaims{
			IssuedAt:  time.Now().Unix(),
			ExpiresAt: time.Now().Add(30 * time.Minute).Unix(),
		},
	})
	accessToken, err := token.SignedString(s.key)
	if err != nil {
		return "", "", err
	}
	return accessToken, userID, nil
}

func (s *Secretary) GetTokenForUser(userID string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &modelclaims.MyCustomClaims{
		UserID: userID,
		StandardClaims: jwt.StandardClaims{
			IssuedAt:  time.Now().Unix(),
			ExpiresAt: time.Now().Add(30 * time.Minute).Unix(),
		},
	})
	return token.SignedString(s.key)
}
