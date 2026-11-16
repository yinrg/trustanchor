//go:build windows

package truststore

import (
	"bytes"
	"crypto/x509"
	"errors"
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const storeLocation = `LocalMachine\Root（受信任的根证书颁发机构）`

var crypt32 = windows.NewLazySystemDLL("crypt32.dll")

var procCertAddEncodedCertificateToStore = crypt32.NewProc("CertAddEncodedCertificateToStore")

func openMachineRoot(readonly bool) (windows.Handle, error) {
	name, err := windows.UTF16PtrFromString("ROOT")
	if err != nil {
		return 0, err
	}
	flags := uint32(windows.CERT_SYSTEM_STORE_LOCAL_MACHINE)
	if readonly {
		flags |= windows.CERT_STORE_READONLY_FLAG
	}
	store, err := windows.CertOpenStore(
		windows.CERT_STORE_PROV_SYSTEM_W,
		0,
		0,
		flags,
		uintptr(unsafe.Pointer(name)),
	)
	if err != nil {
		return 0, fmt.Errorf("打开本机受信任根证书存储失败: %w", err)
	}
	return store, nil
}

// CertEnumCertificatesInStore 在枚举结束时返回 CRYPT_E_NOT_FOUND（系统存储）
// 或 ERROR_NO_MORE_FILES（外部存储），两者都表示「没有更多证书」，不是错误。
var errNoMoreCerts = syscall.Errno(windows.CRYPT_E_NOT_FOUND)

func findCertificate(store windows.Handle, der []byte) (*windows.CertContext, error) {
	var prev *windows.CertContext
	for {
		ctx, err := windows.CertEnumCertificatesInStore(store, prev)
		if ctx == nil {
			if err == nil || errors.Is(err, windows.ERROR_NO_MORE_FILES) || errors.Is(err, errNoMoreCerts) {
				return nil, nil
			}
			return nil, err
		}
		prev = ctx
		if int(ctx.Length) == len(der) && bytes.Equal(unsafe.Slice(ctx.EncodedCert, int(ctx.Length)), der) {
			return ctx, nil
		}
	}
}

func queryStatus(_ *x509.Certificate, der []byte) (State, error) {
	store, err := openMachineRoot(true)
	if err != nil {
		return State{}, err
	}
	defer windows.CertCloseStore(store, 0)

	ctx, err := findCertificate(store, der)
	if err != nil {
		return State{}, err
	}
	if ctx == nil {
		return State{Location: storeLocation}, nil
	}
	_ = windows.CertFreeCertificateContext(ctx)
	return State{Installed: true, Location: storeLocation}, nil
}

func installCert(der []byte) error {
	store, err := openMachineRoot(false)
	if err != nil {
		return err
	}
	defer windows.CertCloseStore(store, 0)

	if ctx, err := findCertificate(store, der); err != nil {
		return err
	} else if ctx != nil {
		_ = windows.CertFreeCertificateContext(ctx)
		return nil
	}

	r1, _, e1 := procCertAddEncodedCertificateToStore.Call(
		uintptr(store),
		uintptr(windows.X509_ASN_ENCODING|windows.PKCS_7_ASN_ENCODING),
		uintptr(unsafe.Pointer(&der[0])),
		uintptr(len(der)),
		uintptr(windows.CERT_STORE_ADD_REPLACE_EXISTING),
		0,
	)
	if r1 == 0 {
		return fmt.Errorf("写入受信任根证书存储失败: %w", e1)
	}
	return nil
}

func uninstallCert(der []byte) error {
	store, err := openMachineRoot(false)
	if err != nil {
		return err
	}
	defer windows.CertCloseStore(store, 0)

	ctx, err := findCertificate(store, der)
	if err != nil {
		return err
	}
	if ctx == nil {
		return ErrNotInstalled
	}
	if err := windows.CertDeleteCertificateFromStore(ctx); err != nil {
		return fmt.Errorf("删除根证书失败: %w", err)
	}
	return nil
}
