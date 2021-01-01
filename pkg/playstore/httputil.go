package playstore

import (
	"github.com/gojektech/heimdall/v6/hystrix"
	"net/http"
	"time"
)

func createHTTPClient() (*hystrix.Client, error) {
	return hystrix.NewClient(
		hystrix.WithHTTPTimeout(5*time.Second),
		hystrix.WithMaxConcurrentRequests(10),
		hystrix.WithErrorPercentThreshold(20),
		hystrix.WithRetryCount(5),
	), nil
}

// Sometimes the server returns 404 Not Found when using fresh credentials
// This may be a caching problem, so try retrying few times
func httpDoRetryOnNotFound(httpClient *hystrix.Client, req *http.Request) (res *http.Response, err error) {
	const retryCount = 4

	for i := 0; i < retryCount; i++ {
		res, err = httpClient.Do(req)
		if err != nil {
			return
		}
		if res.StatusCode == 404 {
			time.Sleep(5 * time.Second)
			continue
		}
		return
	}
	return
}
