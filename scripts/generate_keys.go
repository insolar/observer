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
	"github.com/insolar/insolar/keystore"
	"github.com/insolar/insolar/log"
	"github.com/insolar/insolar/platformpolicy"
	"github.com/spf13/cobra"
)

var (
	_outputDir  string
	_debugLevel string
	_keysFile   = "keys.json"
)

func main() {
	_parseInputParams()
	_mustMakeDir(_outputDir)

	KeysPath := filepath.Join(_outputDir, _keysFile)
	genCertificate(KeysPath, false)
}

func _parseInputParams() {
	var rootCmd = &cobra.Command{}

	rootCmd.Flags().StringVarP(
		&_outputDir, "output", "o", _baseDir(), "output directory")
	rootCmd.Flags().StringVarP(
		&_debugLevel, "debuglevel", "d", "Debug", "debug level")

	err := rootCmd.Execute()
	_check("Wrong input params:", err)
}

func (g *certGen) generateKeys() {
	privKey, err := g.keyProcessor.GeneratePrivateKey()
	checkError("Failed to generate private key:", err)
	pubKey := g.keyProcessor.ExtractPublicKey(privKey)
	fmt.Println("Generate keys")
	g.pubKey, g.privKey = pubKey, privKey
}

func (g *certGen) loadKeys() {
	keyStore, err := keystore.NewKeyStore(g.keysFileOut)
	checkError("Failed to laod keys", err)

	g.privKey, err = keyStore.GetPrivateKey("")
	checkError("Failed to GetPrivateKey", err)

	fmt.Println("Load keys")
	g.pubKey = g.keyProcessor.ExtractPublicKey(g.privKey)
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

func (g *certGen) writeCertificate(cert []byte) {
	f, err := openFile(g.certFileOut)
	checkError("Failed to open file with certificate:", err)

	_, err = f.Write(cert)
	checkError("Failed to write file with certificate:", err)

	fmt.Println("Write certificate to", g.certFileOut)
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

	rootKeysFile string
	API          string

	keysFileOut string
	certFileOut string

	pubKey  crypto.PublicKey
	privKey crypto.PrivateKey
}

func genCertificate(
	// rootKeysFile string,
	// url string,
	keysFile string,
	// certFile string,
	reuseKeys bool,
) {

	g := &certGen{
		keyProcessor: platformpolicy.NewKeyProcessor(),
		// rootKeysFile: rootKeysFile,
		// API:          url,
		keysFileOut: keysFile,
		// certFileOut:  certFile,
	}

	g.generateKeys()

	if !reuseKeys {
		g.writeKeys()
	}
	// g.writeCertificate(cert)
}

func _mustMakeDir(dir string) {
	err := os.MkdirAll(dir, 0775)
	_check("couldn't create directory "+dir, err)
	fmt.Println("generate_insolar_configs.go: creates dir", dir)
}

func _baseDir() string {
	return defaults.LaunchnetDir()
}

func _check(msg string, err error) {
	if err == nil {
		return
	}

	logCfg := configuration.NewLog()
	logCfg.Formatter = "text"
	inslog, _ := log.NewLog(logCfg)
	inslog.WithField("error", err).Fatal(msg)
}
