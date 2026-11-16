package truststore

import (
	"crypto/sha256"
	"crypto/x509"
	_ "embed"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"time"
)

//go:embed rootCA.crt
var embeddedPEM []byte

var (
	ErrNotElevated   = errors.New("需要管理员权限才能修改系统信任库")
	ErrCancelled     = errors.New("已取消管理员授权")
	ErrNoCertificate = errors.New("未内嵌有效的根证书：请用 PEM 格式的根证书替换 internal/truststore/rootCA.crt 后重新构建")
	ErrNotInstalled  = errors.New("根证书未安装")
)

type Info struct {
	Subject     string    `json:"subject"`
	Fingerprint string    `json:"fingerprint"`
	NotBefore   time.Time `json:"not_before"`
	NotAfter    time.Time `json:"not_after"`
	IsCA        bool      `json:"is_ca"`
}

type State struct {
	Info
	Installed bool   `json:"installed"`
	Location  string `json:"location,omitempty"`
}

func Status() (State, error) {
	cert, der, err := certificate()
	if err != nil {
		return State{}, err
	}
	st, err := queryStatus(cert, der)
	if err != nil {
		return State{}, err
	}
	st.Info = describe(cert, der)
	return st, nil
}

func Install() (State, error) {
	if _, der, err := certificate(); err != nil {
		return State{}, err
	} else if !IsElevated() {
		return State{}, ErrNotElevated
	} else if err := installCert(der); err != nil {
		return State{}, err
	}
	return Status()
}

func Uninstall() (State, error) {
	if _, der, err := certificate(); err != nil {
		return State{}, err
	} else if !IsElevated() {
		return State{}, ErrNotElevated
	} else if err := uninstallCert(der); err != nil {
		return State{}, err
	}
	return Status()
}

func certificate() (*x509.Certificate, []byte, error) {
	block, _ := pem.Decode(embeddedPEM)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, nil, ErrNoCertificate
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("解析内嵌根证书失败: %w", err)
	}
	if !cert.IsCA {
		return nil, nil, errors.New("内嵌证书缺少 basicConstraints CA:TRUE，不能作为根证书")
	}
	if err := cert.CheckSignatureFrom(cert); err != nil {
		return nil, nil, fmt.Errorf("内嵌证书不是自签根证书: %w", err)
	}
	return cert, block.Bytes, nil
}

func describe(cert *x509.Certificate, der []byte) Info {
	sum := sha256.Sum256(der)
	parts := make([]string, len(sum))
	for i, b := range sum {
		parts[i] = fmt.Sprintf("%02X", b)
	}
	return Info{
		Subject:     cert.Subject.String(),
		Fingerprint: strings.Join(parts, ":"),
		NotBefore:   cert.NotBefore,
		NotAfter:    cert.NotAfter,
		IsCA:        cert.IsCA,
	}
}
