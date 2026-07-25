package jwt

import "errors"

var ErrInvalidInput = errors.New("auth/jwt: invalid input")
var ErrTokenExpired = errors.New("auth/jwt: token expired")
var ErrInvalidToken = errors.New("auth/jwt: invalid token")
