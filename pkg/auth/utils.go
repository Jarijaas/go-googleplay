package auth

import (
	"bufio"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	xhttp "github.com/Jarijaas/go-tls-exposed/http"
	xtls "github.com/Jarijaas/go-tls-exposed/tls"
	log "github.com/sirupsen/logrus"
	"io"
	"math/big"
	"net/url"
	"strings"
)

const GooglePubkey = "AAAAgMom/1a/v0lblO2Ubrt60J2gcuXSljGFQXgcyZWveWLEwo6prwgi3iJIZdodyhKZQrNWp5nKJ3srRXcUW+F1BD3baEVGcmEgqaLZUNBjm057pKRI16kB0YppeGx5qIQ5QjKzsR8ETQbKLNWgRY0QRNVz34kMJR3P/LgHax/6rmf5AAAAAwEAAQ=="

func parseKeyValues(r io.Reader) map[string]string {
	scanner := bufio.NewScanner(r)

	kvs :=  map[string]string{}

	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), "=")
		kvs[strings.ToLower(strings.TrimSuffix(parts[0], "="))] = parts[1]
	}
	return kvs
}

// Encrypt creds using RSA and google pub key
// https://github.com/NoMore201/googleplay-api/blob/master/gpapi/googleplay.py
// If randSrc is nill, uses crypto/rand.Reader
func encryptCredentials(email string, password string, randSrc *io.Reader) (string, error) {

	if randSrc == nil {
		randSrc = &rand.Reader
	}

	pubKeyBin, err := base64.StdEncoding.DecodeString(GooglePubkey)
	if err != nil {
		return "", err
	}

	modulusLen := binary.BigEndian.Uint32(pubKeyBin)

	modulus := pubKeyBin[4:modulusLen + 4]

	offset := modulusLen + 4

	exponentLen := binary.BigEndian.Uint32(pubKeyBin[offset:offset + 4])

	exponentBytes := make([]byte, 4)
	copy(exponentBytes[4 - exponentLen:], pubKeyBin[offset + 4:])

	exponent := int(binary.BigEndian.Uint32(exponentBytes)) // 65537

	hash := sha1.New()
	hash.Write(pubKeyBin)
	digest := hash.Sum(nil)

	h := append([]byte{}, 0x00)
	h = append(h, digest[:4] ...)

	n := new(big.Int)
	n.SetBytes(modulus)

	pubKey := &rsa.PublicKey{
		N: n,
		E: exponent,
	}

	msg := append([]byte(email), 0x00)
	msg = append(msg, []byte(password) ...)

	ciphertext, err := rsa.EncryptOAEP(sha1.New(), *randSrc, pubKey, msg, nil)
	if err != nil {
		return "", err
	}

	final := append(h, ciphertext ...)

	return base64.URLEncoding.EncodeToString(final), nil
}

// Create http client that bypasses the TLS fingerprint check
// Uses modified tls package, so may be insecure
// Therefore, use this only when necessary
func createXTLSHttpClient() *xhttp.Client {
	conf := &xtls.Config{
		CipherSuites: []uint16{
			0x1302,			0x1303,			0x1301,			0xc02c,
			0xc030,			0xc02b,			0xc02f,			0xcca9,
			0xcca8,			0x00a3,			0x009f,			0x00a2,
			0x009e,			0xccaa,			0xc0af, 		0xc0ad,
			0xc024,			0xc028,			0xc00a, 		0xc014,
			0xc0a3,			0xc09f,			0x006b, 		0x006a,
			0x0039,			0x0038,			0xc0ae,			0xc0ac,
			0xc023,			0xc027,			0xc009,			0xc013,
			0xc0a2,			0xc09e,			0x0067,			0x0040,
			0x0033,			0x0032,			0x009d,			0x009c,
			0xc0a1,			0xc09d,			0xc0a0,			0xc09c,
			0x003d,			0x003c,			0x0035, 		0x002f,
			0x00ff,
		},
		TicketSupported: true,
		PskModes: []uint8{xtls.PskModeDHE},
		SupportedVersions: []uint16{xtls.VersionTLS13, xtls.VersionTLS12},
		SupportedSignatureAlgorithms: []xtls.SignatureScheme{
			0x0403, 0x0503, 0x0603, 0x0807, 0x0808, 0x0809, 0x080a, 0x080b, 0x0804, 0x0805,
			0x0806, 0x0401, 0x0501, 0x0601, 0x0303, 0x0301, 0x0302, 0x0402, 0x0502, 0x0602,
		},
		OscpStapling: true,
		Scts: true,
		CompressionMethods: []uint8{xtls.CompressionNone},
		SecureRenegotiationSupported: false,
		ClientHelloVersion: xtls.VersionTLS12,
		SupportedPoints: []uint8{xtls.PointFormatUncompressed, 1, 2},
		SupportedCurves: []xtls.CurveID{0x001d, 0x0017, 0x001e, 0x0019, 0x0018},
		Extensions: []uint16{
			xtls.ExtensionServerName, xtls.ExtensionSupportedPoints, xtls.ExtensionSupportedCurves,
			xtls.ExtensionSessionTicket, xtls.ExtensionEncryptThenMac, xtls.ExtensionExtendedMasterSecret,
			xtls.ExtensionSignatureAlgorithms, xtls.ExtensionSupportedVersions, xtls.ExtensionSignatureAlgorithmsCert,
			xtls.ExtensionPSKModes, xtls.ExtensionKeyShare,
		},
	}

	transport := &xhttp.Transport{
		TLSClientConfig:        conf,
	}

	return &xhttp.Client{Transport: transport}
}

func getSubToken(masterToken string) (string, error) {

	params := url.Values{}
	params.Set("service", "androidmarket")
	// params.Set("app", "com.android.vending")
	params.Set("Token", masterToken)
	/*params.Set("token_request_options", "CAA4AQ==")
	params.Set("system_partition", "1")
	params.Set("_opt_is_called_from_account_manager", "1")*/


	/*params.Set("source", "android")
	params.Set("client_sig", "38918a453d07199354f8b19af05ec6562ced5788")
	params.Set("callerSig", "38918a453d07199354f8b19af05ec6562ced5788")
	params.Set("lang", "fi")
	params.Set("device_country", "fi")
	params.Set("has_permission", "1")*/

	httpClient := createXTLSHttpClient()

	req, err := xhttp.NewRequest("POST", AuthURL, strings.NewReader(params.Encode()))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	/*req.Header.Set("device", gsfId)
	req.Header.Set("app", "com.android.vending")*/

	res, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}

	kvs := parseKeyValues(res.Body)

	errorDesc, has := kvs["error"]
	if has {
		return "", fmt.Errorf("google auth API returned error: %s", errorDesc)
	}

	log.Debugf("Round token results: %v", kvs)
	return kvs["auth"], nil
}

func getPlayStoreAuthSubToken(email string, encryptedPasswd string) (string, error) {

	params := url.Values{}
	params.Set("service", "androidmarket")
	// params.Set("app", "com.android.vending")

	params.Set("Email", email)
	params.Set("EncryptedPasswd", encryptedPasswd)
	params.Set("add_account", "1")

	/*params.Set("source", "android")
	params.Set("client_sig", "38918a453d07199354f8b19af05ec6562ced5788")
	params.Set("callerSig", "38918a453d07199354f8b19af05ec6562ced5788")
	params.Set("lang", "fi")
	params.Set("device_country", "fi")
	params.Set("has_permission", "1")*/

	httpClient := createXTLSHttpClient()

	req, err := xhttp.NewRequest("POST", AuthURL, strings.NewReader(params.Encode()))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}

	kvs := parseKeyValues(res.Body)

	errorDesc, has := kvs["error"]
	if has {
		log.Debugf("Error response contents: %v", kvs)
		return "", fmt.Errorf("google auth API returned error: %s, info: %s", errorDesc, kvs["info"])
	}

	masterToken, has := kvs["token"]
	if !has {
		return "", fmt.Errorf("AuhSubToken response does not have token: %v", kvs)
	}

	log.Debugf("Got master token: %s", masterToken)
	return getSubToken(masterToken)
}

func boolP(value bool) *bool {
	return &value
}

func intP(value int32) *int32 {
	return &value
}

func int64P(value int64) *int64 {
	return &value
}

func stringP(value string) *string {
	return &value
}