//
// Copyright 2019 Insolar Technologies GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package insconfig

import (
	goflag "flag"
	"fmt"
	"reflect"
	"strings"

	"github.com/mitchellh/mapstructure"
	flag "github.com/spf13/pflag"

	"github.com/insolar/insolar/log"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

type ConfigStruct interface {
	GetConfig() interface{}
}

type Params struct {
	ConfigStruct ConfigStruct
	EnvPrefix    string
	// For go flags compatibility
	GoFlags *goflag.FlagSet
	// For spf13/pflags compatibility
	PFlags     *flag.FlagSet
	ViperHooks []mapstructure.DecodeHookFunc
}

func Load(params Params) (ConfigStruct, error) {
	if params.EnvPrefix == "" {
		return nil, errors.New("EnvPrefix should be defined")
	}
	if params.GoFlags != nil {
		flag.CommandLine.AddGoFlagSet(params.GoFlags)
	}
	if params.PFlags != nil {
		flag.CommandLine.AddFlagSet(params.PFlags)
	}
	var configPath = flag.String("config", "", "path to config")
	flag.Parse()

	return load(params, configPath)
}

func load(params Params, path *string) (ConfigStruct, error) {
	v := viper.New()

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.SetEnvPrefix(params.EnvPrefix)

	v.SetConfigFile(*path)
	if err := v.ReadInConfig(); err != nil {
		return nil, errors.Wrapf(err, "failed to load config")
	}
	actual := params.ConfigStruct.GetConfig()
	params.ViperHooks = append(params.ViperHooks, mapstructure.StringToTimeDurationHookFunc(), mapstructure.StringToSliceHookFunc(","))
	err := v.UnmarshalExact(actual, viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(
		params.ViperHooks...,
	)))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal config file into configuration structure")
	}
	if err := checkAllValuesIsSet(v, params.ConfigStruct); err != nil {
		return nil, err
	}

	return actual.(ConfigStruct), nil
}

func checkAllValuesIsSet(v *viper.Viper, c interface{}) error {
	names := deepFieldNames(c, "")
	for _, val := range names {
		if !v.IsSet(val) {
			return errors.New(fmt.Sprintf("Value not found in config: %s", val))
		}
	}
	return nil
}

func deepFieldNames(iface interface{}, prefix string) []string {
	names := make([]string, 0)
	ifv := reflect.ValueOf(iface)

	for i := 0; i < ifv.NumField(); i++ {
		v := ifv.Field(i)

		switch v.Kind() {
		case reflect.Struct:
			subPrefix := ""
			if prefix != "" {
				subPrefix = prefix + "."
			}
			names = append(names, deepFieldNames(v.Interface(), subPrefix+ifv.Type().Field(i).Name)...)
		default:
			names = append(names, prefix+"."+ifv.Type().Field(i).Name)
		}
	}

	return names
}

// todo clean password
func PrintConfig(c ConfigStruct) {
	cc := &c
	out, err := yaml.Marshal(cc)
	if err != nil {
		log.Error(errors.Wrapf(err, "failed to marshal default config structure"))
	}
	log.Infof("Loaded configuration: \n %s \n", string(out))
}

// func cleanSecrets(c *ConfigStruct) *ConfigStruct {
// 	cleanedConfig := *c
// 	cleanedConfig.DB.URL = replaceDBPassword(cleanedConfig.DB.URL)
// 	return &cleanedConfig
// }

// func replaceDBPassword(url string) string {
// 	re := regexp.MustCompile(`^(?P<start>.*)(:(?P<pass>[^@\/:?]+)@)(?P<end>.*)$`)
// 	var result []byte
// 	if re.MatchString(url) {
// 		for _, submatches := range re.FindAllStringSubmatchIndex(url, -1) {
// 			result = re.ExpandString(result, `$start:<masked>@$end`, url, submatches)
// 		}
// 		return string(result)
// 	}
// 	return url
// }
