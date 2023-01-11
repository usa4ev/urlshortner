package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/google/uuid"
)

const key string = "9cc1ee455a3363ffc504f40006f70d0c8276648a5d3eb3f9524e94d1b7a83aef"

type SessionStoreLoader interface {
	LoadUser(session string) (string, error)
	StoreSession(id, session string) error
}

// OpenSession return new userID & token for a new session
func OpenSession(s SessionStoreLoader) (string, string, error) {
	usrID := uuid.New().String()
	openToken, err := generateRandom(16)
	if err != nil {
		return "", "", fmt.Errorf("failed to create token for user ID: %v \n%v", usrID, err.Error())
	}

	token, err := sealToken(openToken)
	if err != nil {
		return "", "", err
	}
	err = s.StoreSession(usrID, openToken)
	if err != nil {
		return "", "", err
	}

	return usrID, token, nil
}

func LoadUser(token string, s SessionStoreLoader) (string, error) {
	openToken, err := unsealToken(token)
	if err != nil {
		log.Printf("failed to open passed token %v", openToken)
	}

	userId, err := s.LoadUser(openToken)
	if err != nil {
		return "", err
	}

	return userId, nil
}

func generateRandom(size int) (string, error) {
	b := make([]byte, size)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}

func unsealToken(token string) (string, error) {
	hexID, err := hex.DecodeString(token)
	if err != nil {
		return "", err
	}

	k, err := hex.DecodeString(key)
	if err != nil {
		return "", err
	}

	aesblock, err := aes.NewCipher(k[:32])
	if err != nil {
		return "", err
	}

	aesgcm, err := cipher.NewGCM(aesblock)
	if err != nil {
		return "", err
	}

	nonce := k[len(k)-aesgcm.NonceSize():]

	dst, err := aesgcm.Open(nil, nonce, hexID, nil)
	if err != nil {
		return "", err
	}

	return string(dst), err
}

func sealToken(usrID string) (string, error) {
	k, err := hex.DecodeString(key)
	if err != nil {
		return "", err
	}

	aesblock, err := aes.NewCipher(k[:32])
	if err != nil {
		return "", err
	}

	aesgcm, err := cipher.NewGCM(aesblock)
	if err != nil {
		return "", err
	}

	nonce := k[len(k)-aesgcm.NonceSize():]

	dst := aesgcm.Seal(nil, nonce, []byte(usrID), nil)

	return hex.EncodeToString(dst), err
}
