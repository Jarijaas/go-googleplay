package keyring

import (
	"log"
	"testing"
)

func TestKeyringToken(t *testing.T)  {
	tokenIn := "test-token"

	err := SaveToken(GSFID, tokenIn)
	if err != nil {
		t.Fatal(err)
	}

	tokenOut, err := GetToken(GSFID)
	if err != nil {
		t.Fatal(err)
	}

	if tokenIn != tokenOut {
		t.Fatalf("Token saved to keyring (%s) is not equal to the token retrieved from the keyring (%s)",
			tokenIn, tokenOut)
	}

	err = DeleteToken(GSFID)
	if err != nil {
		log.Fatalf("Could not delete auth token: %v", err)
	}
}
