package main

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/insolar/insolar/instrumentation/inslogger"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/insolar/observer/configuration"
)

func main() {
	for filePath, cfg := range configuration.Configurations() {
		out, _ := yaml.Marshal(cfg)
		err := ioutil.WriteFile(filePath, out, 0644)
		if err != nil {
			log := inslogger.FromContext(context.Background())
			log.Error(errors.Wrapf(err, "failed to write config file"))
			return
		}
		fmt.Println(filePath)
	}
}
