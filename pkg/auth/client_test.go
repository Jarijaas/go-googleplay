package auth

import (
	"io"
	"math/rand"
	"os"
	"strings"
	"testing"
)

const EncryptedCreds =
	"AFcb4KQ3_C42um8WqcTQstcPXz6KAzqV9ScldV9iDnqHtNMntBiUcz5-X3nb2w3BhtlAla7mOb" +
		"0fUve69X5LbTPYW_Dn5Hp3XKNEMwrt11K2OpldeN-htRz2hRgjpz1qv9VsVlWGN763dZeW6dIs9MjFAbFg7Ucq9KDXtBelilxbFJQm8Q=="

func TestCredentialsEncryption(t *testing.T) {
	rng := rand.New(rand.NewSource(0))
	rdr := io.Reader(rng)

	encrypted, err := encryptCredentials("example@example.org", "pass123", &rdr)

	if err != nil {
		t.Errorf("encryptCredentials returned error: %v", err)
	}

	if encrypted != EncryptedCreds {
		t.Error("encryptedCredentials result does not match")
	}

	t.Log(encrypted)
}

func TestKeyValueParser(t *testing.T) {
	r := strings.NewReader("LSID=BAD_COOKIE\r\nAuth=123")
	kvs := parseKeyValues(r)

	if kvs["Auth"] != "123" {
		t.Errorf("Key value is incorrect: %s", kvs["Auth"])
	}
}

func TestAuthenticationE2E(t *testing.T) {
	email := os.Getenv("GPLAY_EMAIL")
	password := os.Getenv("GPLAY_PASSWORD")

	if len(email) == 0 || len(password) == 0 {
		t.Skipf("GooglePlay account email or password is not configured, skip authentication E2E test")
	}

	client, err := CreatePlaystoreAuthClient(&Config{
		Email:     email,
		Password:     password,
	})
	if err != nil {
		t.Error(err)
	}

	err = client.Authenticate()
	if err != nil {
		t.Error(err)
	}

}