package cmd

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(loginCmd)
}

var loginCmd = &cobra.Command{
	Use: "login",
	Short: "Login using the credentials, returns new or cached gsfId and authSub",
	RunE: func(cmd *cobra.Command, args []string) error {
		gplay, err := createPlaystoreClient()
		if err != nil {
			return err
		}

		auth := gplay.GetAuthClient()
		err = auth.Authenticate()
		if err != nil {
			return err
		}

		log.Infof("GPLAY_GSFID=%s", auth.GetGsfId())
		log.Infof("GPLAY_AUTHSUB=%s", auth.GetAuthSubToken())
		return nil
	},
}