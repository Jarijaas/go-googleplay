# Google Playstore Client

Binaries are [here](https://github.com/Jarijaas/go-googleplay/releases)

Terminology:

* Google Services Framework ID (GSFID) - Device specific identifier (hex string)
    * Generated against a device config send to Google servers
* AuthSub Token - Authentication token to Playstore, exchanged against account email and password

## CLI Usage

```
Client for Google Playstore, can download apps

Usage:
  gplay [flags]
  gplay [command]

Available Commands:
  download    Download app
  help        Help about any command
  login       Login using the credentials, returns new or cached gsfId and authSub

Flags:
      --authSub string    Alternatively, set env var GPLAY_AUTHSUB
      --email string
      --force-login       Authenticate, even if current gsfId and authSubToken are valid
      --gsfId string      Alternatively, set env var GPLAY_GSFID
  -h, --help              help for gplay
      --password string
  -v, --verbose           Enable debug messages

Use "gplay [command] --help" for more information about a command.
```

For example, to download whatsapp:
```
gplay download --id com.whatsapp --out whatsapp.apk
```

## API Usage

To download a file to disk:

````go
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
````

This project is based on [NoMore201/googleplay-api](https://github.com/NoMore201/googleplay-api) GNU General Public License
