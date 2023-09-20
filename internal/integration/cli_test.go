// +build !node

package integration

import (
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/go-pg/pg"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/internal/testutils"
)

var (
	db        *pg.DB
	pgOptions pg.Options
)

func TestStatsCollector(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		output, err := runCommand("stats-collector", "--config=./.artifacts/stats-collector.yaml")
		require.NotContains(t, output, "error")
		require.NoError(t, err, "error with output: %s", output)
	})
}

func TestMain(m *testing.M) {
	err := os.Chdir("../..")
	if err != nil {
		log.Fatalf("could not change dir: %v", err)
	}

	cmd := exec.Command("make", "build", "config")
	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("%s\n\ncould not make binary: %v", string(output), err)
	}

	var dbCleaner func()
	db, pgOptions, dbCleaner = testutils.SetupDB("./scripts/migrations/")

	retCode := m.Run()
	dbCleaner()
	os.Exit(retCode)
}

func runCommand(cmdName string, args ...string) (string, error) {
	cmd := exec.Command("./bin/"+cmdName, args...)
	cmd.Env = append(
		os.Environ(),
		strings.ToUpper(strings.Replace(cmdName, "-", "", -1))+"_DB_URL=postgres://"+pgOptions.User+":"+pgOptions.Password+"@"+
			pgOptions.Addr+"/"+pgOptions.Database+"?sslmode=disable",
	)
	output, err := cmd.CombinedOutput()
	return string(output), err
}
