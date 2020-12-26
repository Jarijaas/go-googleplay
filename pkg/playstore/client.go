package playstore

import (
	"errors"
	"fmt"
	"github.com/gojektech/heimdall/v6/hystrix"
	"github.com/golang/protobuf/proto"
	"github.com/jarijaas/go-gplayapi/pkg/auth"
	"github.com/jarijaas/go-gplayapi/pkg/common"
	"github.com/jarijaas/go-gplayapi/pkg/playstore/pb"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

const (
	FDFEUrl = common.APIBaseURL + "/fdfe/"
	SearchUrl = FDFEUrl + "search"
	TocUrl = FDFEUrl + "toc"
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

func (client *Client) get(url string) (*pb.ResponseWrapper, error) {
	// Do auth if needed
	if !client.authClient.HasAuthToken() {
		if err := client.authClient.Authenticate(); err != nil {
			return nil, err
		}
	}

	httpClient, err := createHTTPClient()
	if err != nil {
		return nil, err
	}

	searchReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	searchReq.Header.Set("X-DFE-Device-Id", client.authClient.GetGsfId())
	searchReq.Header.Set("Authorization", fmt.Sprintf(
		"GoogleLogin auth=%s", client.authClient.GetAuthSubToken()))

	searchReqRes, err := httpClient.Do(searchReq)
	if err != nil {
		return nil, err
	}

	log.Infof("http status: %s", searchReqRes.Status)

	data, err := ioutil.ReadAll(searchReqRes.Body)

	// fmt.Print(string(data))

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

func (client *Client) Search(query string) (*pb.SearchResponse, error) {

	// c param is content type, 0=book global?, 1=book, 3=app, 4=video

	resWrap, err := client.get(fmt.Sprintf("%s?c=3&q=%s", SearchUrl, query))
	if err != nil {
		return nil, err
	}
	return resWrap.Payload.SearchResponse, err

	/*log.Infof("searchResponse: %v", searchRes)

	for _, doc := range searchRes.Doc {
		if doc == nil {
			continue
		}

		log.Infof("Found docId: %s", *doc.Docid)

		if doc.Title != nil {
			log.Infof("Doc name: %s", *doc.Title)
		}

		for _, child := range doc.Child {
			if child == nil {
				continue
			}
			log.Infof("Found child doc: %s", *child.Docid)
		}
	}

	log.Infof("Next page url: %s", searchRes.GetNextPageUrl())

	return nil*/
}


// DownloadFile downloads a file and write it to disk during download
// https://golangcode.com/download-a-file-from-a-url/
/*func DownloadFile(urlPath string, dir string, filename string) error {
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

func createHTTPClient() (*hystrix.Client, error) {
	return hystrix.NewClient(
		hystrix.WithHTTPTimeout(5 * time.Second),
		hystrix.WithMaxConcurrentRequests(10),
		hystrix.WithErrorPercentThreshold(20),
	), nil
}

/**
Get app details by its package name
 */
func (client *Client) GetDetails(packageName string) {



}

/**
Download app from playstore by its package name

In order to download the app, the app is "purchased" first
If `versionCode` is nil, downloads the latest version
 */
func (client *Client) Download(packageName string, versionCode int) (io.Reader, error)  {

	searchRes, err := client.Search("posti")
	if err != nil {
		return nil, err
	}


	for _, doc := range searchRes.Doc {
		if doc == nil {
			continue
		}
		// log.Infof("doc: %v", doc)

		for _, child := range doc.Child {
			if child == nil {
				continue
			}
			log.Infof("child: %s", *child.Title)
		}

	}

	return nil, nil
}

/**
Check if the client has valid auth creds to the playstore
 */
func (client *Client) IsValidAuthToken() bool {
	_, err := client.Search("")
	return err == nil
}