package detection

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
)

type language_dotnet interface {
	Parse_dotnet(string)
}

func Parse_dotnet(path string) string {
	version := detectVersion3(path)
	return version
}

func detectVersion3(path string) string {
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
