package cmd

import (
	"github.com/jarijaas/go-gplayapi/pkg/device"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"path"
)

func init() {
	rootCmd.AddCommand(deviceCmd)
	deviceCmd.AddCommand(cloneCmd)
	deviceCmd.AddCommand(listProfilesCmd)
}

var deviceCmd = &cobra.Command{
	Use: "device",
	Short: "Device operations",
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

		filepath := path.Join("./", profile.PreferredFilename())

		log.Infof("Save device profile to %s", filepath)
		return profile.SaveToFile(filepath)
	},
}

var listProfilesCmd = &cobra.Command{
	Use: "profiles",
	Short: "List known device profiles",
	Run: func(cmd *cobra.Command, args []string) {
		for _, name := range device.GetBundledProfileNames() {
			log.Info(name)
		}
	},
}