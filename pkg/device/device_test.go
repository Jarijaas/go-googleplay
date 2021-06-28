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
	names := GetBundledProfileNames()
	if len(names) == 0 {
		t.Fatal("Could not find bundled profiles")
	}
}