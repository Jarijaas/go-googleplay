package device

import (
	"embed"
	"fmt"
	"github.com/jarijaas/go-gplayapi/pkg/adb"
	"io/fs"
	"os"
	"strconv"
	"strings"
)

//go:embed profiles/**.ini
var bundledProfiles embed.FS

func GetBundledProfileNames() []string {
	var names []string
	_ = fs.WalkDir(bundledProfiles, "profiles", func(p string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		names = append(names, strings.TrimSuffix(strings.TrimPrefix(p, "profiles/"), ".ini"))
		return nil
	})
	return names
}

type Profile struct {
	props         map[string][]string
	propOverrides map[string][]string
}

func LoadProfileFromFile(filename string) *Profile {
	return &Profile{}
}

func LoadProfileFromADBGetPropOutput(output string) *Profile {
	return &Profile{
		props: parseGetPropOutput(output),
	}
}

func LoadProfileFromAttachedUSBDevice() (*Profile, error) {
	// Todo: improve adb device selection logic
	cli, err := adb.CreateClient()
	if err != nil {
		return nil, err
	}

	cli.SelectAnyUsbDevice()

	rawProps, err := cli.Device().RunCommand("getprop")
	if err != nil {
		return nil, err
	}

	fmt.Print(rawProps)
	return LoadProfileFromADBGetPropOutput(rawProps), nil
}

func (profile *Profile) GetStringPropValue(key string) string {
	values, _ := profile.props[key]
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func (profile *Profile) GetIntPropValue(key string) (int, error) {
	values, _ := profile.props[key]
	if len(values) == 0 {
		return 0, fmt.Errorf("prop not found")
	}
	return strconv.Atoi(values[0])
}

func (profile *Profile) UserFriendlyName() string {
	return profile.GetStringPropValue("ro.product.marketname")
}

func (profile *Profile) Brand() string {
	return profile.GetStringPropValue("ro.product.brand")
}

func (profile *Profile) Model() string {
	return profile.GetStringPropValue("ro.product.model")
}

func (profile *Profile) DeviceName() string {
	return profile.GetStringPropValue("ro.product.device")
}

func (profile *Profile) SdkVer() int {
	ver, _ := profile.GetIntPropValue("ro.build.version.sdk")
	return ver
}

func (profile *Profile) Manufacturer() string {
	return profile.GetStringPropValue("ro.product.manufacturer")
}

func makeStringFilenameFriendly(val string) string {
	return strings.ReplaceAll(strings.ToLower(val), " ", "_")
}

func (profile *Profile) PreferredFilename() string {
	return fmt.Sprintf("%s_%s_sdk_%d.ini",
		makeStringFilenameFriendly(profile.Manufacturer()), makeStringFilenameFriendly(profile.DeviceName()),
		profile.SdkVer())
}

func (profile *Profile) SaveToFile(filepath string) error {
	f, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(convertPropsToIniFormat(profile.props))
	return err
}
