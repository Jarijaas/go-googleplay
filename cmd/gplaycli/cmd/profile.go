package cmd

import (
	"github.com/jarijaas/go-gplayapi/pkg/config"
	"github.com/jarijaas/go-gplayapi/pkg/device"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"path"
)

func init() {
	rootCmd.AddCommand(profileCmd)
	profileCmd.AddCommand(cloneCmd)
	profileCmd.AddCommand(listProfilesCmd)
}

var profileCmd = &cobra.Command{
	Use: "profile",
	Short: "Profile operations",
}

var cloneCmd = &cobra.Command{
	Use: "clone",
	Short: "Clones the profile of the attached USB device",
	Long: `Clones the profile of the attached USB device using "adb shell getprops" and save it into a file for later use.
Tries to filter out sensitive values. Review the file before releasing it publicly.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, err := device.LoadProfileFromAttachedUSBDevice()
		if err != nil {
			return err
		}

		filepath := path.Join(config.GetConfigDirectoryProfilesPath(), profile.PreferredFilename())
		log.Infof("Save device profile to %s", filepath)
		return profile.SaveToFile(filepath)
	},
}

var listProfilesCmd = &cobra.Command{
	Use: "list",
	Short: "List known device profiles",
	Run: func(cmd *cobra.Command, args []string) {
		for _, prof := range device.GetAllProfiles() {
			log.Info(prof.Name)
		}
	},
}