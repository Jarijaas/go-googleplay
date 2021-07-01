package device

import (
	"bytes"
	"embed"
	"fmt"
	"github.com/jarijaas/go-gplayapi/pkg/adb"
	"github.com/jarijaas/go-gplayapi/pkg/config"
	log "github.com/sirupsen/logrus"
	"gopkg.in/ini.v1"
	"io"
	"io/fs"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

//go:embed profiles/**.ini
var bundledProfiles embed.FS

const (
	PropScreenHeight  = "wm.screen.height"
	PropScreenWidth   = "wm.screen.width"
	PropScreenDensity = "ro.sf.lcd_density"
	PropABIList       = "ro.vendor.product.cpu.abilist"
	PropTimezone      = "persist.sys.timezone"
	PropLocale        = "persist.sys.locale"
	PropGsfVersion 	  = "gsf.version"
)

type ProfileFile struct {
	Name string
	Path string
	bundled bool
}

func GetBundledProfiles() []ProfileFile {
	var profiles []ProfileFile
	_ = fs.WalkDir(bundledProfiles, "profiles", func(p string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		profiles = append(profiles, ProfileFile{
			Name: strings.TrimSuffix(filepath.Base(p), ".ini"),
			Path: p,
			bundled: true,
		})
		return nil
	})
	return profiles
}

func GetConfigDirProfiles() ([]ProfileFile, error) {
	var profiles []ProfileFile

	matches, err := filepath.Glob(path.Join(config.GetConfigDirectoryPath(), "profiles", "**.ini"))
	if err != nil {
		return profiles, err
	}

	for _, match := range matches {
		profiles = append(profiles, ProfileFile{
			Name: strings.TrimSuffix(filepath.Base(match), ".ini"),
			Path: match,
			bundled: false,
		})
	}
	return profiles, nil
}

func GetAllProfiles() []ProfileFile{
	var profiles []ProfileFile

	profiles = append(profiles, GetBundledProfiles()...)

	configDirProfiles, err := GetConfigDirProfiles()
	if err != nil {
		log.Warnf("Could not load config dir profiles: %v", err)
	}
	return append(profiles, configDirProfiles...)
}


func ReadBundledProfile(profile ProfileFile) ([]byte, error) {
	return bundledProfiles.ReadFile(profile.Path)
}

type Profile struct {
	props         map[string][]string
	propOverrides map[string][]string
}

func LoadProfileFromIniData(data []byte) (*Profile, error) {
	cfg, err := ini.Load(data)
	if err != nil {
		return nil, err
	}

	if len(cfg.Sections()) == 0 {
		return nil, fmt.Errorf("ini file does not have sections")
	}
	sect := cfg.Sections()[0]

	props := map[string][]string{}
	for _, key := range sect.Keys() {
		props[key.Name()] = strings.Split(key.Value(), ",")
	}

	return &Profile{
		props: props,
	}, nil
}

func LoadProfileFromADBGetPropOutput(output string) *Profile {
	return &Profile{
		props: parseGetPropOutput(output),
	}
}

func LoadDefaultProfile() (*Profile, error) {
	name := GetBundledProfiles()[0]
	data, err := ReadBundledProfile(name)
	if err != nil {
		return nil, err
	}
	return LoadProfileFromIniData(data)
}

func LoadProfile(profile ProfileFile) (*Profile, error) {
	var data []byte
	var err error

	if profile.bundled {
		data, err = ReadBundledProfile(profile)
		if err != nil {
			return nil, err
		}
	} else {
		data, err = ioutil.ReadFile(profile.Path)
		if err != nil {
			return nil, err
		}
	}
	return LoadProfileFromIniData(data)
}

const pmDumpVersionCodeRegex = "(?m:versionCode=([0-9]+))"

func parsePMDumpVersionCode(data string) int {
	r := regexp.MustCompile(pmDumpVersionCodeRegex)

	match := r.FindStringSubmatch(data)
	if len(match) != 2 {
		return 0
	}

	version, err := strconv.Atoi(match[1])
	if err != nil {
		return 0
	}
	return version
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
	profile := LoadProfileFromADBGetPropOutput(rawProps)

	// get screen dimensions
	wmOutput, err := cli.Device().RunCommand("wm", "size")
	if err != nil {
		return nil, err
	}

	/*
	Get screen information
	 */
	wmOutput = strings.TrimSpace(strings.TrimPrefix(wmOutput, "Physical size: "))

	log.Infof("Screen: %s", wmOutput)

	dims := strings.Split(wmOutput, "x")
	width := dims[0]
	height := dims[1]

	profile.SetProp(PropScreenHeight, height)
	profile.SetProp(PropScreenWidth, width)

	/**
	Get Google Services version information
	 */
	gsfInfo, err := cli.Device().RunCommand("pm", "dump", "com.google.android.gms")
	if err != nil {
		return nil, err
	}

	gsfVersion := parsePMDumpVersionCode(gsfInfo)
	profile.SetProp(PropGsfVersion, strconv.Itoa(gsfVersion))

	return profile, nil
}

func (profile *Profile) GetStringPropValue(key string, defaultValue string) string {
	values, _ := profile.props[key]
	if len(values) == 0 {
		return defaultValue
	}
	return values[0]
}

func (profile *Profile) GetStringPropArrValue(key string) []string {
	values, _ := profile.props[key]
	return values
}

func (profile *Profile) GetIntPropValue(key string, defaultValue int) int {
	values, _ := profile.props[key]
	if len(values) == 0 {
		return defaultValue
	}

	value, err := strconv.Atoi(values[0])
	if err != nil {
		return defaultValue
	}
	return value
}

func (profile *Profile) UserFriendlyName() string {
	return profile.GetStringPropValue("ro.product.marketname", "unknown")
}

func (profile *Profile) Brand() string {
	return profile.GetStringPropValue("ro.product.brand", "unknown")
}

func (profile *Profile) Model() string {
	return profile.GetStringPropValue("ro.product.model", "unknown")
}

func (profile *Profile) DeviceName() string {
	return profile.GetStringPropValue("ro.product.device", "unknown")
}

func (profile *Profile) SdkVer() int32 {
	return int32(profile.GetIntPropValue("ro.build.version.sdk", 29))
}

func (profile *Profile) BuildProduct() string {
	return profile.GetStringPropValue("ro.build.product", "unknown")
}

func (profile *Profile) BuildFingerprint() string {
	return profile.GetStringPropValue("ro.build.fingerprint", "unknown")
}

func (profile *Profile) Manufacturer() string {
	return profile.GetStringPropValue("ro.product.manufacturer", "unknown")
}

func (profile *Profile) ProductDevice() string {
	return profile.GetStringPropValue("ro.product.device", "unknown")
}

func (profile *Profile) ProductModel() string {
	return profile.GetStringPropValue("ro.product.model", "unknown")
}

func (profile *Profile) GoogleServicesVer() int32 {
	return int32(profile.GetIntPropValue(PropGsfVersion, 204713063)) // Google Play Services Version: 20.47.13
}

func (profile *Profile) GlEsVersion() int32 {
	return int32(profile.GetIntPropValue("ro.opengles.version", 196610))
}

func (profile *Profile) ScreenWidth() int32 {
	return int32(profile.GetIntPropValue(PropScreenWidth, 0))
}

func (profile *Profile) ScreenHeight() int32 {
	return int32(profile.GetIntPropValue(PropScreenHeight, 0))
}

func (profile *Profile) ScreenDensity() int32 {
	return int32(profile.GetIntPropValue(PropScreenDensity, 0))
}

func (profile *Profile) NativePlatforms() []string {
	return profile.GetStringPropArrValue(PropABIList)
}

func makeStringFilenameFriendly(val string) string {
	return strings.ReplaceAll(strings.ToLower(val), " ", "_")
}

func (profile *Profile) PreferredFilename() string {
	return fmt.Sprintf("%s_%s_sdk_%d.ini",
		makeStringFilenameFriendly(profile.Manufacturer()), makeStringFilenameFriendly(profile.DeviceName()),
		profile.SdkVer())
}

func saveFile(r io.Reader, filepath string) error {
	dirPath := path.Dir(filepath)
	err := os.MkdirAll(dirPath,os.ModePerm)
	if err != nil {
		return err
	}

	f, err := os.Create(filepath)
	if err != nil {
		return err
	}

	_, err = io.Copy(f, r)
	return err
}

func (profile *Profile) SaveToFile(filepath string) error {
	return saveFile(bytes.NewReader(convertPropsToIniFormat(profile.props)),  filepath)
}

func (profile *Profile) Locale() string {
	return profile.GetStringPropValue(PropLocale, "en-US")
}

func (profile *Profile) Timezone() string {
	return profile.GetStringPropValue(PropTimezone, "America/New_York")
}

func (profile *Profile) BuildId() string {
	return profile.GetStringPropValue("ro.build.id", "unknown")
}

func (profile *Profile) SetProp(key string, value string) {
	profile.props[key] = []string{value}
}

func (profile *Profile) BuildIdHardware() string {
	return profile.GetStringPropValue("ro.build.id.hardware", "unknown")
}

func generateRandomString(charset []rune, length int) string {
	rand.Seed(time.Now().UTC().UnixNano())

	val := make([]rune, length)
	for i, _  := range val {
		val[i] = charset[rand.Intn(len(charset))]
	}
	return string(val)
}

func (profile *Profile) Meid() string {
	return generateRandomString([]rune("0123456789"), 15)
}

func (profile *Profile) MacAddr() string {
	return generateRandomString([]rune("0123456789abcdef"), 12)
}

func (profile *Profile) OtaCert() []string {
	return []string{"lIbs5KNFXmSDFVsGAYhR5r5I/ig="}
}

func (profile *Profile) SerialNumber() string {
	return generateRandomString([]rune("0123456789abcdef"), 12)
}

type SysConfig struct {
	Timezone string
	Locale   string
}

func (profile *Profile) SetSysConfig(config *SysConfig) {
	if config.Timezone != "" {
		profile.SetProp(PropTimezone, config.Timezone)
	}
	if config.Locale != "" {
		profile.SetProp(PropLocale, config.Locale)
	}
}
