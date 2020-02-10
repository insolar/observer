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
	"os"
	"reflect"
	"strings"

	"github.com/mitchellh/mapstructure"
	flag "github.com/spf13/pflag"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

// this should be implemented by local config struct
type ConfigStruct interface {
	GetConfig() interface{}
}

type Params struct {
	ConfigStruct ConfigStruct
	// Prefix for environment variables
	EnvPrefix string
	// Custom viper decoding hooks
	ViperHooks []mapstructure.DecodeHookFunc
}

type ConfigPathGetter interface {
	GetConfigPath() string
}

// Adds "--config" flag and read path from it
type DefaultConfigPathGetter struct {
	// For go flags compatibility
	GoFlags *goflag.FlagSet
	// For spf13/pflags compatibility
	PFlags *flag.FlagSet
}

func (g DefaultConfigPathGetter) GetConfigPath() string {
	if g.GoFlags != nil {
		flag.CommandLine.AddGoFlagSet(g.GoFlags)
	}
	if g.PFlags != nil {
		flag.CommandLine.AddFlagSet(g.PFlags)
	}
	configPath := flag.String("config", "", "path to config")
	flag.Parse()
	return *configPath
}

type insConfigurator struct {
	params           Params
	configPathGetter ConfigPathGetter
}

func NewInsConfigurator(params Params, getter ConfigPathGetter) insConfigurator {
	return insConfigurator{
		params:           params,
		configPathGetter: getter,
	}
}

// Loads configuration from path and making checks
func (i *insConfigurator) Load() (ConfigStruct, error) {
	if i.params.EnvPrefix == "" {
		return nil, errors.New("EnvPrefix should be defined")
	}

	configPath := i.configPathGetter.GetConfigPath()
	return i.load(configPath)
}

func (i *insConfigurator) load(path string) (ConfigStruct, error) {
	// todo extract viper
	v := viper.New()

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.SetEnvPrefix(i.params.EnvPrefix)

	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return nil, errors.Wrapf(err, "failed to load config")
	}
	actual := i.params.ConfigStruct.GetConfig()
	i.params.ViperHooks = append(i.params.ViperHooks, mapstructure.StringToTimeDurationHookFunc(), mapstructure.StringToSliceHookFunc(","))
	err := v.UnmarshalExact(actual, viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(
		i.params.ViperHooks...,
	)))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal config file into configuration structure")
	}
	configStructKeys, err := i.checkAllValuesIsSet(v)
	if err != nil {
		return nil, err
	}

	if err := i.checkNoExtraENVValues(configStructKeys); err != nil {
		return nil, err
	}

	return actual.(ConfigStruct), nil
}

func (i *insConfigurator) checkNoExtraENVValues(structKeys []string) error {
	prefixLen := len(i.params.EnvPrefix)
	for _, e := range os.Environ() {
		if len(e) > prefixLen && e[0:prefixLen]+"_" == strings.ToUpper(i.params.EnvPrefix)+"_" {
			kv := strings.SplitN(e, "=", 2)
			key := strings.ReplaceAll(strings.Replace(strings.ToLower(kv[0]), i.params.EnvPrefix+"_", "", 1), "_", ".")
			found := false
			for _, val := range structKeys {
				if strings.ToLower(val) == key {
					found = true
					break
				}
			}
			if !found {
				return errors.New(fmt.Sprintf("Value not found in config: %s", key))
			}
		}
	}
	return nil
}

func (i *insConfigurator) checkAllValuesIsSet(v *viper.Viper) ([]string, error) {
	names := deepFieldNames(i.params.ConfigStruct, "")
	for _, val := range names {
		if !v.IsSet(val) {
			return nil, errors.New(fmt.Sprintf("Value not found in config: %s", val))
		}
	}
	return names, nil
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
			prefWithPoint := ""
			if prefix != "" {
				prefWithPoint = prefix + "."
			}
			names = append(names, prefWithPoint+ifv.Type().Field(i).Name)
		}
	}

	return names
}

// todo clean password
func (i *insConfigurator) PrintConfig(c ConfigStruct) {
	cc := &c
	out, err := yaml.Marshal(cc)
	if err != nil {
		fmt.Println("failed to marshal default config structure")
	}
	fmt.Printf("Loaded configuration: \n %s \n", string(out))
}
