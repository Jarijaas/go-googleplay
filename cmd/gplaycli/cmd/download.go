package cmd

import (
	"fmt"
	"github.com/cheggaaa/pb/v3"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io"
	"os"
	"path"
)

var (
	appPackageName string
	appVersionCode int
	outApkName string
	outDownloadDir string
)

func init() {
	downloadCmd.Flags().StringVar(&appPackageName, "id", "", "The app package name e.g., \"com.whatsapp\"")
	downloadCmd.Flags().IntVar(&appVersionCode, "version",  0,
		"App version code, latest if not specified")
	downloadCmd.Flags().StringVar(&outApkName, "out", "", "Save APK as")
	downloadCmd.Flags().StringVar(&outDownloadDir, "dir", "./", "Where to download files")

	rootCmd.AddCommand(downloadCmd)
}

var downloadCmd = &cobra.Command{
	Use: "download",
	Short: "Download application apk",
	RunE: func(cmd *cobra.Command, args []string) error {
		gplay, err := createPlaystoreClient()
		if err != nil {
			return err
		}

		log.Debugf("Download %s", appPackageName)

		reader, downloadInfo, err := gplay.Download(appPackageName, appVersionCode)
		if err != nil {
			return err
		}

		bar := pb.Full.Start64(downloadInfo.Size)
		barReader := bar.NewProxyReader(reader)

		if outApkName == "" {
			outApkName = fmt.Sprintf("%s.apk", appPackageName)
		}
		filepath := path.Join(outDownloadDir, outApkName)

		f, err := os.Create(filepath)
		if err != nil {
			return err
		}
		_, err = io.Copy(f, barReader)

		bar.Finish()
		return err
	},
}
