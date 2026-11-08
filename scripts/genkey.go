// Run locally: go run scripts/genkey.go PRIVATE_PATH PUBLIC_PATH
package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
)

func main() {
	if len(os.Args) != 3 {
		panic("private and public paths required")
	}
	if _, e := os.Stat(os.Args[1]); !os.IsNotExist(e) {
		panic("refusing to replace an existing private key")
	}
	k, e := rsa.GenerateKey(rand.Reader, 3072)
	if e != nil {
		panic(e)
	}
	b, e := x509.MarshalPKCS8PrivateKey(k)
	if e != nil {
		panic(e)
	}
	if e = os.WriteFile(os.Args[1], pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: b}), 0600); e != nil {
		panic(e)
	}
	p, e := x509.MarshalPKIXPublicKey(&k.PublicKey)
	if e != nil {
		panic(e)
	}
	if e = os.WriteFile(os.Args[2], pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: p}), 0644); e != nil {
		panic(e)
	}
}
