package detection

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
)

type language1 interface {
	Parse_php(string)
}

func Parse_php(path string) string {
	version := detectVersion(path)
	return version
}

func detectVersion(path string) string {
	details, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Println(err)
	} else {
		var lang_details map[string]string
		err = json.Unmarshal(details, &lang_details)
		if err != nil {
			log.Fatal("Error during Unmarshal(): ", err)
		} else {
			ver := lang_details["version"]
			return ver
		}
	}
	return ""
}
