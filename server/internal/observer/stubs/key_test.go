package stubs

import (
	"testing"

	"github.com/insolar/insolar/platformpolicy"
	"github.com/pkg/errors"
)

func TestP(t *testing.T) {
	key := "-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEFhw9v2vl9OkMedBoKz8GTndZyx5S\n/KHFc3OKOoEhUPZwuNo1q3bXTaeJ1WBcs4MjGBBGuC5w1i3WcNfJHzyyLw==\n-----END PUBLIC KEY-----\n"
	kp := platformpolicy.NewKeyProcessor()
	pubKey, err := kp.ImportPublicKeyPEM([]byte(key))
	if err != nil {
		t.Error(errors.Wrap(err, "failed to import a public key from PEM"))
	}
	t.Logf("pub: %v", pubKey)
}
