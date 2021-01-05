package chezmoi

import (
	"io/ioutil"
	"os"
	"runtime"

	"go.uber.org/multierr"
)

// An EncryptionTool encrypts and decrypts data.
type EncryptionTool interface {
	Decrypt(filenameHint string, ciphertext []byte) ([]byte, error)
	DecryptToFile(filenameHint string, ciphertext []byte) (string, func() error, error)
	Encrypt(plaintext []byte) ([]byte, error)
	EncryptFile(filename string) ([]byte, error)
}

// encryptionToolDecrypt is the default implementation of
// EncryptionTool.Decrypt.
func encryptionToolDecrypt(t EncryptionTool, filenameHint string, ciphertext []byte) (plaintext []byte, err error) {
	filename, cleanup, err := t.DecryptToFile(filenameHint, ciphertext)
	if err != nil {
		return
	}
	defer func() {
		err = multierr.Append(err, cleanup())
	}()
	return ioutil.ReadFile(filename)
}

// encryptionToolEncrypt is the default implementation of
// EncryptionTool.Encrypt.
func encryptionToolEncrypt(t EncryptionTool, pattern string, plaintext []byte) (ciphertext []byte, err error) {
	tempFile, err := ioutil.TempFile("", pattern)
	if err != nil {
		return
	}
	defer func() {
		err = multierr.Append(err, os.RemoveAll(tempFile.Name()))
	}()

	if runtime.GOOS != "windows" {
		if err = tempFile.Chmod(0o600); err != nil {
			return
		}
	}

	if err = ioutil.WriteFile(tempFile.Name(), plaintext, 0o600); err != nil {
		return
	}

	ciphertext, err = t.EncryptFile(tempFile.Name())
	return
}
