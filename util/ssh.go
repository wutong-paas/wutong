package util

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"

	"golang.org/x/crypto/ssh"
)

// GenOpenSSHKeyPair make a pair of private and public keys for SSH access.
// Private Key generated is PEM encoded
// Public key is encoded in the format for inclusion in an OpenSSH authorized_keys file.
func GenOpenSSHKeyPair() ([]byte, []byte, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	rsaSSHPriKeyBytes := pem.EncodeToMemory(privateKeyPEM)

	pub, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, nil, err
	}

	rsaSSHPubKeyBytes := ssh.MarshalAuthorizedKey(pub)

	return rsaSSHPriKeyBytes, rsaSSHPubKeyBytes, nil
}
