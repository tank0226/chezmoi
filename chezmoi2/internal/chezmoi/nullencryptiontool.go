package chezmoi

import "io/ioutil"

// A nullEncryptionTool returns its input unchanged.
type nullEncryptionTool struct{}

func (*nullEncryptionTool) Decrypt(ciphertext []byte) ([]byte, error) {
	return ciphertext, nil
}

func (*nullEncryptionTool) DecryptToFile(filename string, ciphertext []byte) error {
	return ioutil.WriteFile(filename, ciphertext, 0o600)
}

func (*nullEncryptionTool) Encrypt(plaintext []byte) ([]byte, error) {
	return plaintext, nil
}

func (*nullEncryptionTool) EncryptFile(filename string) ([]byte, error) {
	return ioutil.ReadFile(filename)
}
