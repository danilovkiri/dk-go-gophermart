// Package modelclaims provides types for token authorization.

package modelclaims

import "github.com/golang-jwt/jwt"

type MyCustomClaims struct {
	UserID string `json:"userID"`
	jwt.StandardClaims
}
