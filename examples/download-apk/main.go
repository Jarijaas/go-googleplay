package main

import (
	"github.com/jarijaas/go-gplayapi/pkg/auth"
	"github.com/jarijaas/go-gplayapi/pkg/playstore"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
)

func main()  {
	gplay, err := playstore.CreatePlaystoreClient(&playstore.Config{
		AuthConfig: &auth.Config{
			Email:        "",
			Password:     "",
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// Download the latest version
	reader, downloadInfo, err := gplay.Download("com.whatsapp", 0)
	if err != nil {
		log.Fatal(err)
	}

	log.Infof("Download size: %d bytes", downloadInfo.Size)

	f, err := os.Create("./whatsapp.apk")
	if err != nil {
		log.Fatal(err)
	}

	_, err = io.Copy(f, reader)
	if err != nil {
		log.Fatal(err)
	}
}
