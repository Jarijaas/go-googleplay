package device

import (
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"testing"
)

var propPathsWhitelist = []string{
	"ro.product.*", "ro.opengles.*", "ro.system.*", "ro.build.*", "ro.vendor.*",
}

var propPathsBlacklist = []string{
	"*imei*", // try to filter out props that may contain imei
}

func TestParseGetPropsOutput(t *testing.T) {
	data, err := ioutil.ReadFile("./testdata/device.getprops")
	if err != nil {
		log.Fatal(err)
	}

	props := parseGetPropsOutput(string(data))

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