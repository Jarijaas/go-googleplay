package keyring

import "github.com/zalando/go-keyring"

const KeyringService = "go-gplayapi"

const KeyringAuthToken = "auth-token"

func SaveToken(token string) error {
	return keyring.Set(KeyringService, KeyringAuthToken, token)
}

func GetToken() (string, error) {
	return keyring.Get(KeyringService, KeyringAuthToken)
}

func DeleteToken() error {
	return keyring.Delete(KeyringService, KeyringAuthToken)
}


