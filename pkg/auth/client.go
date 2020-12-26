package auth

// Based on https://github.com/NoMore201/googleplay-api/blob/master/gpapi/googleplay.py

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/jarijaas/go-gplayapi/pkg/common"
	"github.com/jarijaas/go-gplayapi/pkg/keyring"
	"github.com/jarijaas/go-gplayapi/pkg/playstore/pb"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"strconv"
)

const (
	AuthURL = common.APIBaseURL + "/auth"
	CheckinURL = common.APIBaseURL + "/checkin"
)

/**
Handles authentication transparently
Decides based on config parameters how authentication should be performed
 */
type Client struct {
	config *Config
	deviceConsistencyToken string
}

type Config struct {
	Email string
	Password string
	GsfId string
	AuthSubToken string
}

func CreatePlaystoreAuthClient(config *Config) (*Client, error) {
	gsfId, authSub, err := keyring.GetGoogleTokens()
	if err == nil && gsfId != "" && authSub != "" {
		log.Tracef("Found GSIF %s and authSub %s tokens from keyring", gsfId, authSub)
		config.GsfId = gsfId
		config.AuthSubToken = authSub
	}
	return &Client{config: config}, nil
}

type Type string

const (
	EmailPassword Type = "email-pass"
	Token         Type = "token"
	Unknown       Type = ""
)

/**
Use email and passwd if set, otherwise use tokens
 */
func (client *Client) getAuthType() Type {
	if client.config.Email != "" && client.config.Password != "" {
		return EmailPassword
	}

	if client.config.GsfId != ""  && client.config.AuthSubToken  != "" {
		return Token
	}
	return Unknown
}

/**
Check if has necessary tokens (GsfId & AuthSub) for authenticated request, does not check if the tokens are valid
 */
func (client *Client) HasAuthToken() bool {
	return client.config.GsfId != "" && client.config.AuthSubToken != ""
}

func (client *Client) GetGsfId() string {
	return client.config.GsfId
}

func (client *Client) GetAuthSubToken() string {
	return client.config.AuthSubToken
}

// Get "androidId", which is a device specific GSF (google services framework) ID
func (client *Client) getGsfId() (string, error) {
	username := "username"

	locale := "fi"
	timezone := "Europe/Helsinki"
	version := int32(3)
	fragment := int32(0)

	lastCheckinMsec := int64(0)
	userNumber := int32(0)
	cellOperator := "22210"
	simOperator := "22210"
	roaming := "mobile-notroaming"

	checkin := pb.AndroidCheckinProto{
		Build:           nil,
		LastCheckinMsec: &lastCheckinMsec,
		Event:           nil,
		Stat:            nil,
		RequestedGroup:  nil,
		CellOperator:    &cellOperator,
		SimOperator:     &simOperator,
		Roaming:         &roaming,
		UserNumber:      &userNumber,
	}

	checkinReq := &pb.AndroidCheckinRequest{
		Imei:                nil,
		Id:                  nil,
		Digest:              nil,
		Checkin:             &checkin,
		DesiredBuild:        nil,
		Locale:              &locale,
		LoggingId:           nil,
		MarketCheckin:       nil,
		MacAddr:             nil,
		Meid:                nil,
		AccountCookie:       nil,
		TimeZone:            &timezone,
		SecurityToken:       nil,
		Version:             &version,
		OtaCert:             nil,
		SerialNumber:        nil,
		Esn:                 nil,
		DeviceConfiguration: nil,
		MacAddrType:         nil,
		Fragment:            &fragment,
		UserName:            &username,
		UserSerialNumber:    nil,
	}


	rawMsg, err := proto.Marshal(checkinReq)
	if err != nil {
		return "", err
	}

	log.Printf("Raw proto msg: %s", hex.Dump(rawMsg))

	resp, err := http.Post(CheckinURL, "application/x-protobuf", bytes.NewReader(rawMsg))
	if err != nil {
		return "", err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	log.Infof("Response: %s", string(body))


	var checkinResp pb.AndroidCheckinResponse
	err = proto.Unmarshal(body, &checkinResp)
	if err != nil {
		return "", err
	}

	rawMsg, err = proto.Marshal(checkinReq)
	if err != nil {
		return "", err
	}

	resp, err = http.Post(CheckinURL, "application/x-protobuf", bytes.NewReader(rawMsg))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("checkin error: %s %d", resp.Status, resp.StatusCode)
	}

	return strconv.FormatUint(*checkinResp.AndroidId, 16), nil
}


func (client *Client) Authenticate() error {
	authType := client.getAuthType()
	if authType == Unknown {
		return fmt.Errorf(
			"could not select authentication type." +
				"Did you specify the email and the password or alternatively GSFID and authSubToken")
	}

	switch authType {
	case EmailPassword:
		encryptedPasswd, err := encryptCredentials(client.config.Email, client.config.Password, nil)
		if err != nil {
			return err
		}

		/*params := url.Values{}
		params.Set("service", "ac2dm")
		params.Set("add_account", "1")
		params.Set("EncryptedPasswd", encryptedPasswd)
		params.Set("Email", client.config.Email)

		httpClient := createXTLSHttpClient()

		res, err := httpClient.PostForm(AuthURL, params)
		if err != nil {
			return err
		}

		kvs := parseKeyValues(res.Body)

		log.Infof("%v", kvs)

		errorDesc, has := kvs["error"]
		if has {
			return fmt.Errorf("google auth API returned error: %s", errorDesc)
		}

		ac2dmToken, has := kvs["auth"]
		if !has {
			return fmt.Errorf("google auth API response did not contain ac2dm token. Response: %v", kvs)
		}
		log.Debugf("Got ac2dm token: %s", ac2dmToken)*/

		client.config.GsfId, err = client.getGsfId()
		if err != nil {
			return err
		}

		client.config.AuthSubToken, err = getPlayStoreAuthSubToken(client.config.Email, encryptedPasswd)
		if err != nil {
			return err
		}

		log.Infof("Got GsfId and AuthSubToken, saving these to keyring")

		err = keyring.SaveToken(keyring.GSFID, client.config.GsfId)
		if err != nil {
			return err
		}

		err = keyring.SaveToken(keyring.AuthSubToken, client.config.AuthSubToken)
		if err != nil {
			return err
		}
	}

	return nil
}
