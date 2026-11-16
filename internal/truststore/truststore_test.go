package truststore

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"strings"
	"testing"
	"time"
)

func testCA(t *testing.T) (*x509.Certificate, []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("生成密钥失败: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Test Root CA", Organization: []string{"Test"}},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("签发证书失败: %v", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("解析证书失败: %v", err)
	}
	return cert, der
}

func TestDescribeFingerprint(t *testing.T) {
	cert, der := testCA(t)
	info := describe(cert, der)

	parts := strings.Split(info.Fingerprint, ":")
	if len(parts) != 32 {
		t.Fatalf("指纹应为 32 组字节，实际 %d", len(parts))
	}
	for _, p := range parts {
		if len(p) != 2 {
			t.Fatalf("指纹分组格式错误: %q", p)
		}
	}
	if !info.IsCA {
		t.Fatal("CA 证书应被识别为 CA")
	}
	if info.Subject == "" {
		t.Fatal("主体不应为空")
	}
}

func TestRunCLIIgnoresAppArgs(t *testing.T) {
	for _, args := range [][]string{nil, {}, {"--help"}, {"--version"}, {"--install"}} {
		if handled, _ := RunCLI(args); handled {
			t.Fatalf("RunCLI(%v) 不应接管启动参数", args)
		}
	}
}

func TestRunCLIHandlesStatus(t *testing.T) {
	if handled, _ := RunCLI([]string{cmdCAStatus}); !handled {
		t.Fatal("RunCLI 应接管 --ca-status")
	}
}
