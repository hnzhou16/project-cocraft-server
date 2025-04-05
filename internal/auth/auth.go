package auth

import "github.com/golang-jwt/jwt/v5"

type UserAuthenticator interface {
	GenerateToken(claims jwt.Claims) (string, error)
	VerifyToken(token string) (*jwt.Token, error)
}
