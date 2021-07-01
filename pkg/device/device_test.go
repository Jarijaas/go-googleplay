package device

import (
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"testing"
)

func TestParseGetPropsOutput(t *testing.T) {
	data, err := ioutil.ReadFile("./testdata/device.getprops")
	if err != nil {
		log.Fatal(err)
	}

	props := parseGetPropOutput(string(data))

	const testProp = "ro.product.brand"
	const validPropValue = "Xiaomi"
	if val, has := props[testProp]; has {
		if val[0] != validPropValue {
			log.Fatalf("%s should be %s, was %s", testProp, validPropValue, val[0])
		}
	} else {
		log.Fatalf("Props should have %s", testProp)
	}
}

func TestGetBundledProfileNames(t *testing.T) {
	profs := GetBundledProfiles()
	if len(profs) == 0 {
		t.Fatal("Could not find bundled profiles")
	}
}

func TestVersionCodeRegexParse(t *testing.T) {
	versionCode := parsePMDumpVersionCode(`
        863da0b com.google.android.gms/.ads.measurement.GmpConversionTrackingBrokerService filter 5fcd566
    versionCode=17785037 minSdk=28 targetSdk=28
    versionName=17.7.85 (100400-253824076)
    signatures=PackageSignatures{7f248cb version:3, signatures:[e3ca78d8], past signatures:[]}
`)

	if versionCode != 17785037 {
		log.Fatalf("invalid version code match: %d", versionCode)
	}
}
