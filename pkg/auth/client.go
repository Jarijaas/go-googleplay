package auth

// Based on https://github.com/NoMore201/googleplay-api/blob/master/gpapi/googleplay.py

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/Jarijaas/go-tls-exposed/http/httputil"
	"github.com/gojektech/heimdall/v6/hystrix"
	"github.com/golang/protobuf/proto"
	"github.com/jarijaas/go-gplayapi/pkg/common"
	"github.com/jarijaas/go-gplayapi/pkg/playstore/pb"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
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
	/*httpClient, err := createHTTPClient()
	if err != nil {
		return nil, err
	}*/
	return &Client{config: config}, nil
}


type AuthType string

const (
	EmailPassword AuthType = "email-pass"
	Unknown AuthType = ""
)

func (client *Client) getAuthType() AuthType {
	if client.config.Email != "" && client.config.Password != "" {
		return EmailPassword
	}
	return Unknown
}

// Get "androidId", which is a device specific GSF (google services framework) ID
func (client *Client) doCheckin(ac2dmToken string) (string, error) {


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

	if checkinResp.AndroidId != nil {
		log.Infof("Got androidId: %d", *checkinResp.AndroidId)
	}

	for _, setting := range checkinResp.Setting {
		log.Infof("Setting: %s, Value: %s", string(setting.Name), string(setting.Value))
	}

	token := checkinResp.DeviceCheckinConsistencyToken
	if token == nil {
		return "", fmt.Errorf("DeviceCheckinConsistencyToken in checkin response does not exist")
	}
	log.Debugf("Got DeviceCheckinConsistencyToken: %s", *token)

	client.deviceConsistencyToken = *token

	checkinReq.Id = checkinResp.AndroidId
	checkinReq.SecurityToken = checkinResp.SecurityToken
	checkinReq.AccountCookie = append(checkinReq.AccountCookie, fmt.Sprintf("[%s]", client.config.Email))
	checkinReq.AccountCookie = append(checkinReq.AccountCookie, ac2dmToken)

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
				"Did you specify the email and the password or alternatively the auth token")
	}

	switch authType {
	case EmailPassword:
		/*encryptedPasswd, err := encryptCredentials(client.config.Email, client.config.Password, nil)
		if err != nil {
			return err
		}

		params := url.Values{}
		params.Set("service", "ac2dm")
		params.Set("add_account", "0")
		params.Set("EncryptedPasswd", encryptedPasswd)
		params.Set("Email", client.config.Email)

		httpClient := createXTLSCHttpClient()

		res, err := httpClient.PostForm(AuthURL, params)
		if err != nil {
			return err
		}

		kvs := parseKeyValues(res.Body)

		errorDesc, has := kvs["error"]
		if has {
			return fmt.Errorf("google auth API returned error: %s", errorDesc)
		}

		ac2dmToken, has := kvs["auth"]
		if !has {
			return fmt.Errorf("google auth API response did not contain ac2dm token. Response: %v", kvs)
		}
		log.Debugf("Got ac2dm token: %s", ac2dmToken)


		client.config.GsfId, err = client.doCheckin(ac2dmToken)
		if err != nil {
			return err
		}

		client.config.AuthSubToken, err = getPlayStoreAuthSubToken(client.config.Email, client.config.GsfId,
			encryptedPasswd)
		if err != nil {
			return err
		}

		log.Infof("Got GsfId and AuthSubToken, saving these to keyring")

		/*err = client.checkTOC()
		if err != nil {
			return err
		}*/



		/*for _, entry := range searchRes.Entry {
			if entry == nil {
				continue
			}

			log.Infof("Found app: %s", *entry.Title)
		}*/
	}

	return nil
}


func createHTTPClient() (*hystrix.Client, error) {
	return hystrix.NewClient(
		hystrix.WithHTTPTimeout(5 * time.Second),
		hystrix.WithMaxConcurrentRequests(10),
		hystrix.WithErrorPercentThreshold(20),
	), nil
}

/*
// FDFEUrl is a base URL for Playstore DFE api
var FDFEUrl = "https://android.clients.google.com/fdfe"

func dfeGetReq(dfeURL string, deviceID string) ([]byte, error) {
	client, err := newHTTPClient()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", dfeURL, nil)
	if err != nil {
		return nil, err
	}

	// Android device ID, required for all DFE api requests, must be a valid device ID
	req.Header.Add("X-DFE-Device-Id", deviceID)

	req.AddCookie(&http.Cookie{Domain: ".google.com", Name: "HSID", Value: "A1-ROoNZ4JkZFEork"})
	req.AddCookie(&http.Cookie{Domain: ".google.com", Name: "SSID", Value: "ADH00LW58WvhDUs-m"})
	req.AddCookie(&http.Cookie{Domain: ".google.com", Name: "SID", Value: "FAew0fFsuiIR9aFikGdgIdSUdwAOj1Ud1v0O7lh4_wRjVRsKiej4TXteVoiVG0fyjULihQ."})

	// Get the data
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}
/*
// GetAppDetails retrieves app details. E.g name, version, description
func GetAppDetails(docID string, deviceID string) (*app_details.AppDetailsMessage, error) {

	detailsURL := fmt.Sprintf("%s/details?doc=%s", FDFEUrl, docID)
	rawProtoMessage, err := dfeGetReq(detailsURL, deviceID)
	if err != nil {
		return nil, err
	}

	details := &app_details.AppDetailsMessage{}

	err = proto.Unmarshal(rawProtoMessage, details)
	if err != nil {
		return nil, err
	}

	return details, nil
}

// GetAppDeliveryManifest downloads application delivery manifest from the playstore
// It contains file sizes and download URLs for each download
func GetAppDeliveryManifest(docID string, versionCode int, deviceID string) (*app_download_manifest.DeliveryManifest, error) {

	manifestURL := fmt.Sprintf("%s/delivery?doc=%s&ot=1&vc=%d&ia=false&fdcf=1&fdcf=2&da=22", FDFEUrl, docID, versionCode)
	rawProtoMessage, err := dfeGetReq(manifestURL, deviceID)
	if err != nil {
		return nil, err
	}

	manifest := &app_download_manifest.DeliveryManifest{}

	err = proto.Unmarshal(rawProtoMessage, manifest)
	if err != nil {
		return nil, err
	}

	return manifest, nil
}

// DownloadFile downloads a file and write it to disk during download
// https://golangcode.com/download-a-file-from-a-url/
func DownloadFile(urlPath string, dir string, filename string) error {
	client, err := newHTTPClient()
	if err != nil {
		return err
	}

	req, err := http.NewRequest("GET", urlPath, nil)
	if err != nil {
		return err
	}

	// Get the data
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	filepath := fmt.Sprintf("%s/%s", dir, filename)

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}*/