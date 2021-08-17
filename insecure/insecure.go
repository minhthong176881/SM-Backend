package insecure

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"os"

	"github.com/joho/godotenv"
)

var (
	// Cert is a self signed certificate
	Cert tls.Certificate
	// CertPool contains the self signed certificate
	CertPool *x509.CertPool
)

func init() {
	var err error
	err = godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file")
	}
	path := os.Getenv("ABSOLUTE_PATH")
	certPEM, err := ioutil.ReadFile(path + `\insecure\cert.pem`)
	if err != nil {
        log.Println("No RSA private key found, generating temp one")
    }
	keyPEM, err := ioutil.ReadFile(path + `\insecure\key.pem`)
	if err != nil {
        log.Println("No RSA private key found, generating temp one")
    }

	Cert, err = tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
	if err != nil {
		log.Fatalln("Failed to parse key pair:", err)
	}
	Cert.Leaf, err = x509.ParseCertificate(Cert.Certificate[0])
	if err != nil {
		log.Fatalln("Failed to parse certificate:", err)
	}

	CertPool = x509.NewCertPool()
	CertPool.AddCert(Cert.Leaf)
}

func GenRSA(bits int) (*rsa.PrivateKey, error) {
    key, err := rsa.GenerateKey(rand.Reader, bits)
    return key, err
}