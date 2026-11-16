//go:build darwin || linux

package truststore

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/pem"
	"os"
	"strings"
)

func IsElevated() bool {
	return os.Geteuid() == 0
}

func pemBytes(der []byte) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

func fingerprintHex(der []byte) string {
	sum := sha256.Sum256(der)
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}

func tempPEMFile(der []byte) (string, func(), error) {
	f, err := os.CreateTemp("", "root-ca-*.crt")
	if err != nil {
		return "", func() {}, err
	}
	path := f.Name()
	cleanup := func() { os.Remove(path) }
	if _, err := f.Write(pemBytes(der)); err != nil {
		f.Close()
		cleanup()
		return "", func() {}, err
	}
	if err := f.Close(); err != nil {
		cleanup()
		return "", func() {}, err
	}
	return path, cleanup, nil
}

func shellJoin(args []string) string {
	quoted := make([]string, len(args))
	for i, a := range args {
		quoted[i] = shellQuote(a)
	}
	return strings.Join(quoted, " ")
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
