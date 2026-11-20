// Package pki crée une petite autorité de certification, pour le
// développement et les tests : certificats de serveur pour TLS, et de client
// pour l'authentification mutuelle (mTLS). En production, les certificats
// viennent d'une vraie autorité (Let's Encrypt, cert-manager, Vault…).
package pki

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"time"
)

// AC est une autorité de certification.
type AC struct {
	cert *x509.Certificate
	cle  *ecdsa.PrivateKey
	der  []byte
}

// NouvelleAC crée une autorité, valable un jour.
func NouvelleAC(nom string) (*AC, error) {
	cle, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	modele := &x509.Certificate{
		SerialNumber:          numeroDeSerie(),
		Subject:               pkix.Name{CommonName: nom},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLenZero:        true, // ne peut signer que des certificats finaux
	}
	der, err := x509.CreateCertificate(rand.Reader, modele, modele, &cle.PublicKey, cle)
	if err != nil {
		return nil, err
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, err
	}
	return &AC{cert: cert, cle: cle, der: der}, nil
}

// Pool renvoie l'ensemble de confiance contenant l'autorité : RootCAs côté
// client, ClientCAs côté serveur.
func (a *AC) Pool() *x509.CertPool {
	pool := x509.NewCertPool()
	pool.AddCert(a.cert)
	return pool
}

// PEM renvoie le certificat de l'autorité, au format PEM.
func (a *AC) PEM() []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: a.der})
}

// Serveur émet un certificat de serveur pour les noms (« localhost ») et
// adresses IP (« 127.0.0.1 ») donnés.
func (a *AC) Serveur(noms ...string) (Certificat, error) {
	return a.emet(noms[0], x509.ExtKeyUsageServerAuth, noms)
}

// Client émet un certificat de client, qui identifie un service ou un
// utilisateur par son nom (CommonName).
func (a *AC) Client(nom string) (Certificat, error) {
	return a.emet(nom, x509.ExtKeyUsageClientAuth, nil)
}

// Certificat est un certificat et sa clé privée, aux formats PEM.
type Certificat struct {
	CertPEM, ClePEM []byte
}

// TLS renvoie le certificat sous la forme attendue par tls.Config.
func (c Certificat) TLS() (tls.Certificate, error) {
	return tls.X509KeyPair(c.CertPEM, c.ClePEM)
}

func (a *AC) emet(nom string, usage x509.ExtKeyUsage, noms []string) (Certificat, error) {
	cle, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return Certificat{}, err
	}
	modele := &x509.Certificate{
		SerialNumber: numeroDeSerie(),
		Subject:      pkix.Name{CommonName: nom},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{usage},
	}
	for _, n := range noms {
		if ip := net.ParseIP(n); ip != nil {
			modele.IPAddresses = append(modele.IPAddresses, ip)
		} else {
			modele.DNSNames = append(modele.DNSNames, n)
		}
	}
	der, err := x509.CreateCertificate(rand.Reader, modele, a.cert, &cle.PublicKey, a.cle)
	if err != nil {
		return Certificat{}, err
	}
	clePKCS8, err := x509.MarshalPKCS8PrivateKey(cle)
	if err != nil {
		return Certificat{}, err
	}
	return Certificat{
		CertPEM: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		ClePEM:  pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: clePKCS8}),
	}, nil
}

// numeroDeSerie tire un numéro aléatoire de 128 bits, comme l'exigent les
// règles des autorités publiques.
func numeroDeSerie() *big.Int {
	n, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	return n
}
