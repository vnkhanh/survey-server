package utils

import (
    "crypto/rand"
    "encoding/base64"
    "errors"

    "golang.org/x/crypto/bcrypt"
)

func GenerateEditToken() (string, error) {
    b := make([]byte, 32)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    return base64.RawURLEncoding.EncodeToString(b), nil
}

func HashEditToken(token string) (string, error) {
    if token == "" {
        return "", errors.New("empty token")
    }
    hash, err := bcrypt.GenerateFromPassword([]byte(token), bcrypt.DefaultCost)
    return string(hash), err
}

func VerifyEditToken(hashed, token string) bool {
    if hashed == "" || token == "" {
        return false
    }
    return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(token)) == nil
}