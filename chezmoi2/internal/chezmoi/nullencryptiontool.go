package chezmoi

import "io/ioutil"

// A nullEncryption returns its input unchanged.
type nullEncryption struct{}

func (*nullEncryption) Decrypt(ciphertext []byte) ([]byte, error) {
	return ciphertext, nil
}

func (*nullEncryption) DecryptToFile(filename string, ciphertext []byte) error {
	return ioutil.WriteFile(filename, ciphertext, 0o600)
}

func (*nullEncryption) Encrypt(plaintext []byte) ([]byte, error) {
	return plaintext, nil
}

func (*nullEncryption) EncryptFile(filename string) ([]byte, error) {
	return ioutil.ReadFile(filename)
}
