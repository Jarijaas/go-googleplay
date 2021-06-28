package device

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"
)

var propPathsWhitelist = []string{
	"ro.product.*", "ro.opengles.*", "ro.system.*", "ro.build.*", "ro.vendor.*",
}

var propPathsBlacklist = []string{
	".*imei.*", // try to filter out props that may contain imei or other sensitive information
	".*serial.*",
	".*iccid.*",
}


func parseGetPropOutput(data string) map[string][]string {
	kv := map[string][]string{}

	entryRegex := regexp.MustCompile(`(?m:^\[(.*)]: \[(.*)])`)

	entries := entryRegex.FindAllStringSubmatch(data, -1)
	for _, entry := range entries {
		key := entry[1]
		value := entry[2]

		isInWhitelist := false
		isInBlacklist := false

		/*
			Filter out props that are not in the whitelist
		*/
		for _, path := range propPathsWhitelist {
			if matched, _ := regexp.MatchString(path, key); matched {
				isInWhitelist = true
				break
			}
		}
		if !isInWhitelist {
			continue
		}

		/*
			Filter out props that are in the blacklist
		*/
		for _, path := range propPathsBlacklist {
			if matched, _ := regexp.MatchString(path, key); matched {
				isInBlacklist = true
				break
			}
		}
		if isInBlacklist {
			continue
		}
		kv[key] = strings.Split(strings.TrimSpace(value), ",")
	}
	return kv
}


func convertPropsToIniFormat(props map[string][]string) []byte {
	buf := bytes.NewBuffer([]byte{})

	for key, values := range props {
		buf.Write([]byte(fmt.Sprintf("%s=%s\n", key, strings.Join(values, ","))))
	}

	return buf.Bytes()
}

func savePropsToFile(props map[string][]string, filename string) error {
	rawIni := convertPropsToIniFormat(props)
	return ioutil.WriteFile(filename, rawIni, 0644)
}
