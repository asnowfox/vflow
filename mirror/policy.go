package mirror

import (
	"encoding/json"
	"fmt"
	"github.com/VerizonDigital/vflow/vlogger"
	"io/ioutil"
	"os"
)

func LoadPolicy(mirrorCfg string) error {
	b, err := ioutil.ReadFile(mirrorCfg)
	if err != nil {
		vlogger.Logger.Printf("No Mirror config file is defined. \n")
		fmt.Printf("No Mirror config file is defined. \n")
		return err
	}
	err = json.Unmarshal(b, &policyConfigs)
	if err != nil {
		vlogger.Logger.Printf("Mirror config file is wrong, exit! \n")
		fmt.Printf("Mirror config file is wrong,exit! \n")
		os.Exit(-1)
		return err
	}
	return nil
}
