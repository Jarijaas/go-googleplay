package keyring

import "github.com/zalando/go-keyring"

const Service = "go-gplayapi"

type TokenType string

const (
	AuthSubToken TokenType = "authsub-token"
	GSFID        TokenType = "gsfid"
)

func SaveToken(tokenType TokenType, token string) error {
	return keyring.Set(Service, string(tokenType), token)
}

func GetToken(tokenType TokenType) (string, error) {
	return keyring.Get(Service, string(tokenType))
}

func DeleteToken(tokenType TokenType) error {
	return keyring.Delete(Service, string(tokenType))
}

/**
Get GSFID (Google Services ID) and AuthSub token from the keyring
*/
func GetGoogleTokens() (gsid string,  authSub string, err error) {
	authSub, err = GetToken(AuthSubToken)
	if err != nil {
		return
	}
	gsid, err = GetToken(GSFID)
	return
}

