package device

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"
)

func parseGetPropsOutput(data string) map[string][]string {
	beginBrackets := strings.Split(data, "[")

	kv := map[string][]string{}

	// 2 [ and ] brackets per key,value pair
	for i := 0; i < len(beginBrackets) - 2; i += 2 {
		key := beginBrackets[i + 1]
		key = key[:strings.Index(key, "]: ")]
		value := beginBrackets[i + 2]

		value = value[:strings.Index(value, "]")] // ends with closing bracket and new line
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