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
	"crypto"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/insolar/insolar/configuration"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/defaults"
	"github.com/insolar/insolar/log"
	"github.com/insolar/insolar/platformpolicy"
	"github.com/spf13/cobra"
)

var (
	outputDir  string
	debugLevel string
	keysFile   = "keys.json"
)

func main() {
	parseInputParams()
	mustMakeDir(outputDir)

	KeysPath := filepath.Join(outputDir, keysFile)
	genCertificate(KeysPath, false)
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

func (g *certGen) generateKeys() {
	privKey, err := g.keyProcessor.GeneratePrivateKey()
	checkError("Failed to generate private key:", err)
	pubKey := g.keyProcessor.ExtractPublicKey(privKey)
	fmt.Println("Generate keys")
	g.pubKey, g.privKey = pubKey, privKey
}

func (g *certGen) writeKeys() {
	privKeyStr, err := g.keyProcessor.ExportPrivateKeyPEM(g.privKey)
	checkError("Failed to deserialize private key:", err)

	pubKeyStr, err := g.keyProcessor.ExportPublicKeyPEM(g.pubKey)
	checkError("Failed to deserialize public key:", err)

	result, err := json.MarshalIndent(map[string]interface{}{
		"private_key": string(privKeyStr),
		"public_key":  string(pubKeyStr),
	}, "", "    ")
	checkError("Failed to serialize file with private/public keys:", err)

	f, err := openFile(g.keysFileOut)
	checkError("Failed to open file with private/public keys:", err)

	_, err = f.Write(result)
	checkError("Failed to write file with private/public keys:", err)

	fmt.Println("Write keys to", g.keysFileOut)
}

func checkError(msg string, err error) {
	if err != nil {
		fmt.Println(msg, ": ", err)
		os.Exit(1)
	}
}

func openFile(path string) (io.Writer, error) {
	return os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
}

type certGen struct {
	keyProcessor insolar.KeyProcessor
	API          string

	keysFileOut string

	pubKey  crypto.PublicKey
	privKey crypto.PrivateKey
}

func genCertificate(
	keysFile string,
	reuseKeys bool,
) {

	g := &certGen{
		keyProcessor: platformpolicy.NewKeyProcessor(),
		keysFileOut:  keysFile,
	}

	g.generateKeys()

	if !reuseKeys {
		g.writeKeys()
	}
}

func mustMakeDir(dir string) {
	err := os.MkdirAll(dir, 0775)
	check("couldn't create directory "+dir, err)
	fmt.Println("generate_configs.go: creates dir", dir)
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
