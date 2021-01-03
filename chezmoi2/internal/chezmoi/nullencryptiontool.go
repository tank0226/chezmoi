package chezmoi

import "io/ioutil"

// A nullEncryptionTool returns its input unchanged.
type nullEncryptionTool struct{}

func (*nullEncryptionTool) Decrypt(filenameHint string, ciphertext []byte) ([]byte, error) {
	return ciphertext, nil
}

func (*nullEncryptionTool) DecryptToFile(filenameHint string, ciphertext []byte) (string, func() error, error) {
	return filenameHint, func() error { return nil }, nil
}

func (*nullEncryptionTool) Encrypt(plaintext []byte) ([]byte, error) {
	return plaintext, nil
}

func (*nullEncryptionTool) EncryptFile(filename string) ([]byte, error) {
	return ioutil.ReadFile(filename)
}
