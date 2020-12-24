package cmd

import (
	"github.com/spf13/cobra"
)

var (
	appPackageName string
	appVersionCode int
	outFilepath string
)

func init() {
	downloadCmd.Flags().StringVar(&appPackageName, "id", "", "The app package name e.g., \"com.whatsapp\"")
	downloadCmd.Flags().IntVarP(&appVersionCode, "version", "v", 0,
		"App version code, latest if not specified")
	downloadCmd.Flags().StringVar(&outFilepath, "out", "", "Where to download the app")

	rootCmd.AddCommand(downloadCmd)
}

var downloadCmd = &cobra.Command{
	Use: "download",
	Short: "Download app",
	RunE: func(cmd *cobra.Command, args []string) error {
		gplay, err := createPlaystoreClient()
		if err != nil {
			return err
		}

		_, err = gplay.Download(appPackageName, appVersionCode)
		if err != nil {
			return err
		}

		// ioutil.WriteFile("", bytes)

		return err
	},
}
