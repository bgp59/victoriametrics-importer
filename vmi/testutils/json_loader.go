// load json test cases

package vmi_testutils

import (
	"encoding/json"
	"fmt"
	"os"
)

func LoadJsonFile(fileName string, obj any) error {
	fileIo, err := os.Open(fileName)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(fileIo)

	err = decoder.Decode(obj)
	if err != nil {
		return fmt.Errorf("%v: error decoding %#v into %T", err, fileName, obj)
	}
	return nil
}
