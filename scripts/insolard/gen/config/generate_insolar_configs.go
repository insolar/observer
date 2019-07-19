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

package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/insolar/insolar/configuration"
	"github.com/insolar/insolar/insolar/defaults"
	"github.com/insolar/insolar/log"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var (
	defaultOutputConfigNameTmpl = "insolard.yaml"
	defaultHost                 = "127.0.0.1"
	defaultJaegerEndPoint       = ""
	nodeDataDirectoryTemplate   = "data"
	nodeCertificatePathTemplate = "cert.json"
	keysFile                    = "keys.json"

	prometheusConfigTmpl = "scripts/prom/server.yml.tmpl"
	prometheusFileName   = "prometheus.yaml"

	insolardDefaultsConfig = "scripts/insolard/defaults/insolard.yaml"
)

var (
	outputDir  string
	debugLevel string
)

func main() {
	parseInputParams()

	mustMakeDir(outputDir)

	promVars := &promConfigVars{
		Jobs: map[string][]string{},
	}

	nodeDataDirectoryTemplate = filepath.Join(outputDir, nodeDataDirectoryTemplate)
	nodeCertificatePathTemplate = filepath.Join(outputDir, nodeCertificatePathTemplate)

	conf := newDefaultInsolardConfig()

	conf.Host.Transport.Address = fmt.Sprintf("%s:63846", defaultHost)
	conf.Host.Transport.Protocol = "TCP"

	conf.APIRunner.Address = fmt.Sprintf(defaultHost+":191%02d", 7)
	conf.Metrics.ListenAddress = fmt.Sprintf(defaultHost+":80%02d", 7)

	conf.Tracer.Jaeger.AgentEndpoint = defaultJaegerEndPoint
	conf.Log.Level = debugLevel
	conf.Log.Adapter = "zerolog"
	conf.Log.Formatter = "json"

	conf.KeysPath = filepath.Join(outputDir, keysFile)
	conf.Ledger.Storage.DataDirectory = nodeDataDirectoryTemplate
	conf.CertificatePath = nodeCertificatePathTemplate

	writeInsolardConfig(outputDir, conf)

	promVars.addTarget("heavy_observer", conf)

	writePromConfig(promVars)
}

func parseInputParams() {
	var rootCmd = &cobra.Command{}

	rootCmd.Flags().StringVarP(
		&outputDir, "output", "o", baseDir(), "output directory")
	rootCmd.Flags().StringVarP(
		&debugLevel, "debuglevel", "d", "Debug", "debug level")

	err := rootCmd.Execute()
	check("Wrong input params:", err)
}

func newDefaultInsolardConfig() configuration.Configuration {
	holder := configuration.NewHolderWithFilePaths(insolardDefaultsConfig).MustInit(true)
	return holder.Configuration
}

func writeInsolardConfig(dir string, conf configuration.Configuration) {
	fmt.Println("generate_insolar_configs.go: writeInsolardConfigs...")
	data, err := yaml.Marshal(conf)
	check("Can't Marshal insolard config", err)

	fileName := defaultOutputConfigNameTmpl
	fileName = filepath.Join(dir, fileName)
	err = createFileWithDir(fileName, string(data))
	check("failed to create insolard config: "+fileName, err)
}

func createFileWithDir(file string, text string) error {
	mustMakeDir(filepath.Dir(file))
	return makeFile(file, text)
}

type promConfigVars struct {
	Jobs map[string][]string
}

func (pcv *promConfigVars) addTarget(name string, conf configuration.Configuration) {
	jobs := pcv.Jobs
	addrPair := strings.SplitN(conf.Metrics.ListenAddress, ":", 2)
	addr := "host.docker.internal:" + addrPair[1]
	jobs[name] = append(jobs[name], addr)
}

func writePromConfig(pcv *promConfigVars) {
	templates, err := template.ParseFiles(prometheusConfigTmpl)
	check("Can't parse template: "+prometheusConfigTmpl, err)

	var b bytes.Buffer
	err = templates.Execute(&b, pcv)
	check("Can't process template: "+prometheusConfigTmpl, err)

	err = makeFileWithDir(outputDir, prometheusFileName, b.String())
	check("Can't makeFileWithDir: "+prometheusFileName, err)
}

func makeFileWithDir(dir string, name string, text string) error {
	err := os.MkdirAll(dir, 0775)
	if err != nil {
		return err
	}
	file := filepath.Join(dir, name)
	return makeFile(file, text)
}

func makeFile(name string, text string) error {
	fmt.Println("generate_insolar_configs.go: write to file", name)
	return ioutil.WriteFile(name, []byte(text), 0644)
}

func mustMakeDir(dir string) {
	err := os.MkdirAll(dir, 0775)
	check("couldn't create directory "+dir, err)
	fmt.Println("generate_insolar_configs.go: creates dir", dir)
}

func baseDir() string {
	return defaults.LaunchnetDir()
}

func check(msg string, err error) {
	if err == nil {
		return
	}

	logCfg := configuration.NewLog()
	logCfg.Formatter = "text"
	inslog, _ := log.NewLog(logCfg)
	inslog.WithField("error", err).Fatal(msg)
}
