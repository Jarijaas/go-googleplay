package cmd

import (
	"fmt"
	"github.com/jarijaas/go-gplayapi/pkg/auth"
	"github.com/jarijaas/go-gplayapi/pkg/playstore"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
	"os"
	log "github.com/sirupsen/logrus"
)

var (
	email string
	password string
	gsfId string
	authSub string
)

var rootCmd = &cobra.Command{
	Use:   "gplay",
	Short: "This allows browsing the Google Playstore including downloading apps",
	Run: func(cmd *cobra.Command, args []string) {
		// Do Stuff Here
	},
}

func Execute() {

	rootCmd.PersistentFlags().StringVar(&email, "email", "", "")
	rootCmd.PersistentFlags().StringVar(&password, "password", "", "")
	rootCmd.PersistentFlags().StringVar(&gsfId, "gsfId", "", "")
	rootCmd.PersistentFlags().StringVar(&authSub, "authSub", "", "")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
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

	// Ask for creds if not authenticated
	if !gplay.IsValidAuthToken() {
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
	}
	return gplay, err
}