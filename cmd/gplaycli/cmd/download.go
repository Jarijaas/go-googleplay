package cmd

import (
	"github.com/cheggaaa/pb/v3"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
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

		progressCh, err := gplay.DownloadToDisk(appPackageName, appVersionCode, outDownloadDir, outApkName)
		if err != nil {
			return err
		}

		var bar *pb.ProgressBar

		for progress := range progressCh {
			if bar == nil {
				bar = pb.Start64(progress.DownloadSize)
				bar.Set(pb.Bytes, true)
			}

			bar.SetCurrent(progress.DownloadedSize)
			if progress.Err != nil {
				return err
			}
		}

		bar.Finish()

		return err
	},
}
