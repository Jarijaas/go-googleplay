package playstore

import (
	"fmt"
	"github.com/jarijaas/go-gplayapi/pkg/auth"
	"github.com/jarijaas/go-gplayapi/pkg/common"
	"github.com/jarijaas/go-gplayapi/pkg/playstore/pb"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
)

const (
	FDFEUrl = common.APIBaseURL + "/fdfe/"
	SearchUrl = FDFEUrl + "search"
	TocUrl = FDFEUrl + "toc"
)

type Client struct {

	authedClient *auth.Client

}

type Config struct {

	authConfig *auth.Config
}

func CreatePlaystoreClient(config *Config) (*Client, error) {
	authedClient, err := auth.CreatePlaystoreAuthClient(config.authConfig)
	if err != nil {
		return nil, err
	}

	return &Client{
		authedClient: authedClient,
	}, nil
}

func (client *Client) Search(query string) error {

	httpClient2, err := createHTTPClient()
	if err != nil {
		return err
	}

	searchReq, err := http.NewRequest("GET", fmt.Sprintf("%s?c=3&q=%s", SearchUrl, "posti"), nil)
	if err != nil {
		return err
	}

	searchReq.Header.Set("X-DFE-Device-Id", "")
	// searchReq.Header.Set("Authorization", fmt.Sprintf("GoogleLogin auth=%s", client.config.AuthSubToken))
	searchReq.Header.Set("Authorization", "GoogleLogin auth=")

	reqRaw, _ := httputil.DumpRequest(searchReq, true)
	fmt.Print(string(reqRaw))

	searchReqRes, err := httpClient2.Do(searchReq)
	if err != nil {
		return err
	}

	log.Infof("Search http status: %s", searchReqRes.Status)

	data, err := ioutil.ReadAll(searchReqRes.Body)

	// fmt.Print(string(data))

	var responseWrapper pb.ResponseWrapper
	err = proto.Unmarshal(data, &responseWrapper)
	if err != nil {
		return err
	}

	if responseWrapper.Commands != nil && responseWrapper.Commands.DisplayErrorMessage != nil {
		log.Fatal(*responseWrapper.Commands.DisplayErrorMessage)
	}

	searchRes := responseWrapper.Payload.SearchResponse

	log.Infof("searchResponse: %v", searchRes)

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
func (client *Client) Download(packageName string, versionCode *int) (io.Reader, error)  {



}