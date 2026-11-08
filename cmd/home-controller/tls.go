package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Use a separate server identity: the control process must not read LuCI's key.
func renewTLS() error {
	c, err := loadConfig()
	if err != nil {
		return err
	}
	_, binds, err := networkAccess(c)
	if err != nil {
		return err
	}
	host, _ := os.Hostname()
	if !regexpDevice(host) {
		host = "home-controller"
	}
	sans := []string{"DNS:" + host}
	ips := []net.IP{}
	for _, b := range binds {
		h, _, _ := net.SplitHostPort(b)
		if ip := net.ParseIP(h); ip != nil {
			ips = append(ips, ip)
			sans = append(sans, "IP:"+h)
		}
	}
	var cert, key []byte
	if exists("/usr/sbin/router-ca") {
		name := fmt.Sprintf("home-controller-%d-%s", time.Now().Unix(), token()[:8])
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		out, e := exec.CommandContext(ctx, "/usr/sbin/router-ca", "--batch", "--use-algo=ec", "--curve=prime256v1", "--san="+strings.Join(sans, ","), "build-server-full", name, "nopass").CombinedOutput()
		if e != nil {
			return fmt.Errorf("control certificate issuance failed: %v: %s", e, out)
		}
		cert, err = os.ReadFile("/etc/easy-rsa/pki/issued/" + name + ".crt")
		if err != nil {
			return err
		}
		key, err = os.ReadFile("/etc/easy-rsa/pki/private/" + name + ".key")
		if err != nil {
			return err
		}
	} else {
		private, e := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if e != nil {
			return e
		}
		serial, e := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
		if e != nil {
			return e
		}
		t := &x509.Certificate{SerialNumber: serial, Subject: pkix.Name{CommonName: host}, DNSNames: []string{host}, IPAddresses: ips, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().AddDate(1, 0, 0), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
		der, e := x509.CreateCertificate(rand.Reader, t, t, &private.PublicKey, private)
		if e != nil {
			return e
		}
		k, e := x509.MarshalPKCS8PrivateKey(private)
		if e != nil {
			return e
		}
		cert = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
		key = pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: k})
	}
	if _, err = tls.X509KeyPair(cert, key); err != nil {
		return err
	}
	for name, data := range map[string][]byte{"tls.crt": cert, "tls.key": key} {
		f, e := os.CreateTemp(base, ".tls-")
		if e != nil {
			return e
		}
		tmp := f.Name()
		_, e = f.Write(data)
		f.Close()
		if e == nil {
			e = os.Rename(tmp, base+"/"+name)
		}
		os.Remove(tmp)
		if e != nil {
			return e
		}
		adminFile(base + "/" + name)
	}
	return nil
}
