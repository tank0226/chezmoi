package chezmoi

// An EncryptionTool encrypts and decrypts data.
type EncryptionTool interface {
	Decrypt(filenameHint string, ciphertext []byte) ([]byte, error)
	DecryptToFile(filenameHint string, ciphertext []byte) (string, func() error, error)
	Encrypt(plaintext []byte) ([]byte, error)
	EncryptFile(filename string) ([]byte, error)
}
