package cmd

import (
	"fmt"
	"github.com/jarijaas/go-gplayapi/pkg/auth"
	"github.com/jarijaas/go-gplayapi/pkg/playstore"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
	"os"
)

var (
	email string
	password string
	gsfId string
	authSub string
	forceLogin bool
)

var rootCmd = &cobra.Command{
	Use:   "gplay",
	Short: "Allows browsing the Google Playstore, including downloading apps",
	Run: func(cmd *cobra.Command, args []string) {
		// Do Stuff Here
	},
}

func Execute() {
	rootCmd.PersistentFlags().StringVar(&email, "email", "", "")
	rootCmd.PersistentFlags().StringVar(&password, "password", "", "")
	rootCmd.PersistentFlags().StringVar(&gsfId, "gsfId", "", "")
	rootCmd.PersistentFlags().StringVar(&authSub, "authSub", "", "")
	rootCmd.PersistentFlags().BoolVar(&forceLogin, "force-login", false,
		"Authenticate, even if current gsfId and authSubToken are valid")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func createPlaystoreClient() (*playstore.Client, error) {
	authCfg := &auth.Config{
		Email:        email,
		Password:     password,
		GsfId:        gsfId,
		AuthSubToken: authSub,
	}

	gplay, err := playstore.CreatePlaystoreClient(&playstore.Config{
		AuthConfig: authCfg,
	})
	if err != nil {
		return nil, err
	}

	// Force reauthentication by removing current tokens
	if forceLogin {
		authCfg.GsfId = ""
		authCfg.AuthSubToken = ""
	}

	// Ask for creds if not authenticated
	if !gplay.IsValidAuthToken(){
		log.Info("Auth token is not valid, use email and password")

		if authCfg.Email == "" {
			log.Info("Enter email:")
			_, err = fmt.Scanln(&authCfg.Email)
			if err != nil {
				return nil, err
			}
		}

		if authCfg.Password == "" {
			log.Info("Enter password:")
			passwd, err := terminal.ReadPassword(int(os.Stdin.Fd()))
			if err != nil {
				return nil, err
			}
			authCfg.Password = string(passwd)
		}
	} else {
		log.Debugf(
			"Current gsfId and authSubToke are valid. To force reauthentication, use --force-login flag")
	}
	return gplay, err
}