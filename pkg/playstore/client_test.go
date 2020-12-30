package playstore

import (
	"github.com/jarijaas/go-gplayapi/pkg/auth"
	log "github.com/sirupsen/logrus"
	"os"
	"testing"
)

// const TestPackageName = "com.google.android.youtube"
const TestPackageName = "com.whatsapp"

func createPlayStoreTestClient(t *testing.T) *Client {
	gsfId := os.Getenv("GPLAY_GSFID")
	authSub := os.Getenv("GPLAY_AUTHSUB")

	if gsfId == "" || authSub == "" {
		t.Skip("gsfId or authSub is not specified, skip test")
	}

	authConfig := &auth.Config{
		GsfId:        os.Getenv(gsfId),
		AuthSubToken: os.Getenv(authSub),
	}

	client, err := CreatePlaystoreClient(&Config{AuthConfig: authConfig})
	if err != nil {
		t.Fatalf("Could not create playstore client: %v", err)
	}
	return client
}

func TestGetAppDetails(t *testing.T) {
	client := createPlayStoreTestClient(t)

	res, err := client.GetDetails(TestPackageName)
	if err != nil {
		t.Fatalf("Could not get package details: %v", err)
	}

	log.Infof("Got details: %v", res)

	if *res.Docid != TestPackageName {
		t.Fatalf("Package name is incorrect: %s, should be: %s", *res.Docid, TestPackageName)
	}

	/**
	appDetails = self.details(packageName).get('details').get('appDetails')
	            versionCode = appDetails.get('versionCode')
	 */

	log.Infof("App details: %v", res.Details.AppDetails)

	if res.Details.AppDetails.VersionCode == nil {
		t.Fatalf("Version code is not in the response, is your gsfId correct?")
	}
}

func TestDownloadApp(t *testing.T) {
	client := createPlayStoreTestClient(t)

	res, err := client.GetAppDeliveryData(TestPackageName, 0)
	if err != nil {
		t.Fatalf("Could not purchase app: %v", err)
	}

	if res.DownloadUrl == nil {
		t.Fatalf("%s delivery data does not have download URL: %v ", TestPackageName, res)
	}
}