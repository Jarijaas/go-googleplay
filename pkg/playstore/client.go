package playstore

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/jarijaas/go-gplayapi/pkg/auth"
	"github.com/jarijaas/go-gplayapi/pkg/common"
	"github.com/jarijaas/go-gplayapi/pkg/playstore/pb"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strconv"
)

const (
	FDFEUrl     = common.APIBaseURL + "/fdfe/"
	SearchUrl   = FDFEUrl + "search"
	TocUrl      = FDFEUrl + "toc"
	DetailsUrl  = FDFEUrl + "details"
	PurchaseUrl = FDFEUrl + "purchase"
)

type Client struct {
	authClient *auth.Client
}

type Config struct {
	AuthConfig *auth.Config
}

func CreatePlaystoreClient(config *Config) (*Client, error) {
	authedClient, err := auth.CreatePlaystoreAuthClient(config.AuthConfig)
	if err != nil {
		return nil, err
	}

	return &Client{
		authClient: authedClient,
	}, nil
}

func (client *Client) send(url string, bodyParams *url.Values) (*pb.ResponseWrapper, error) {
	// Do auth if needed
	if !client.authClient.HasAuthToken() {
		if err := client.authClient.Authenticate(); err != nil {
			return nil, err
		}
	}

	var body io.Reader

	method := "GET"
	if bodyParams != nil {
		method = "POST"
		body = bytes.NewBufferString(bodyParams.Encode())
	}

	log.Debugf("%s %s", method, url)

	httpClient, err := createHTTPClient()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-DFE-Device-Id", client.authClient.GetGsfId())
	req.Header.Set("Authorization", fmt.Sprintf(
		"GoogleLogin auth=%s", client.authClient.GetAuthSubToken()))

	if bodyParams != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	reqRes, err := httpDoRetryOnNotFound(httpClient, req)
	if err != nil {
		return nil, err
	}
	if reqRes.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected response for %s: %s (%d)",
			url, reqRes.Status, reqRes.StatusCode)
	}

	data, err := ioutil.ReadAll(reqRes.Body)
	if err != nil {
		return nil, err
	}

	var responseWrapper pb.ResponseWrapper
	err = proto.Unmarshal(data, &responseWrapper)
	if err != nil {
		return nil, err
	}

	if responseWrapper.Commands != nil && responseWrapper.Commands.DisplayErrorMessage != nil {
		return &responseWrapper, errors.New(*responseWrapper.Commands.DisplayErrorMessage)
	}
	return &responseWrapper, nil
}


func (client *Client) GetAuthClient() *auth.Client {
	return client.authClient
}

// c param is content type, 0=book global?, 1=book, 3=app, 4=video
func (client *Client) Search(query string) (*pb.SearchResponse, error) {
	resWrap, err := client.send(fmt.Sprintf("%s?c=3&q=%s", SearchUrl, query), nil)
	if err != nil {
		return nil, err
	}
	return resWrap.Payload.SearchResponse, err
}

/**
Get app details by its package name
*/
func (client *Client) GetDetails(packageName string) (*pb.DocV2, error) {
	resWrap, err := client.send(fmt.Sprintf("%s?doc=%s", DetailsUrl, packageName), nil)
	if err != nil {
		return nil, err
	}
	return resWrap.Payload.DetailsResponse.DocV2, nil
}

func (client *Client) Purchase(packageName string, versionCode int) (*pb.BuyResponse, error) {
	params := &url.Values{}
	params.Set("ot", "1")
	params.Set("doc", packageName)
	params.Set("vc", strconv.Itoa(versionCode))

	res, err := client.send(PurchaseUrl, params)
	if err != nil {
		return nil, err
	}
	return res.Payload.BuyResponse, nil
}

/**
Get app delivery data (download URL) for application from playstore

In order to download the app, the app is "purchased" first
If `versionCode` is zero, get delivery data for the latest version
*/
func (client *Client) GetAppDeliveryData(packageName string, versionCode int) (*pb.AndroidAppDeliveryData, error) {
	log.Debugf("Get delivery data for %s", packageName)

	// Get latest version code
	if versionCode == 0 {
		doc, err := client.GetDetails(packageName)
		if err != nil {
			return nil, err
		}
		versionCode = int(*doc.Details.AppDetails.VersionCode)

		log.Debugf("Latest %s version code: %d", packageName, versionCode)
	}

	buyRes, err := client.Purchase(packageName, versionCode)
	if err != nil {
		return nil, err
	}

	purchaseStatusRes := buyRes.PurchaseStatusResponse
	if purchaseStatusRes == nil {
		return nil, fmt.Errorf("response does not contain purchase status response")
	}

	appDeliveryData := purchaseStatusRes.AppDeliveryData
	if appDeliveryData == nil {
		return nil, fmt.Errorf("response does not contain app delivery data")
	}
	return appDeliveryData, nil
}

/**
Download an APK from the playstore to the destination directory

In order to download the app, the app is "purchased" first
If `versionCode` is zero, download the latest version
if `apkName` is "", uses `packageName` as filename
*/
func (client *Client) DownloadToDisk(
	packageName string, versionCode int, downloadDir string, apkName string) (chan DownloadProgress, error) {

	deliveryData, err := client.GetAppDeliveryData(packageName, versionCode)
	if err != nil {
		return nil, err
	}

	if deliveryData.DownloadUrl == nil {
		return nil, fmt.Errorf("deliver data does not contain download url")
	}

	downloadUrl := *deliveryData.DownloadUrl
	log.Debugf("Downloading %s from %s", packageName, downloadUrl)

	checksum, err := base64.RawStdEncoding.DecodeString(*deliveryData.Sha1)
	if err != nil {
		return nil, err
	}

	if apkName == "" {
		apkName = fmt.Sprintf("%s.apk", packageName)
	}

	filepath := path.Join(downloadDir, apkName)
	return downloadFileToDisk(downloadUrl, *deliveryData.DownloadSize, checksum, filepath)
}

/**
Check if the client has valid auth creds to the playstore
*/
func (client *Client) IsValidAuthToken() bool {
	_, err := client.Search("")
	return err == nil
}
