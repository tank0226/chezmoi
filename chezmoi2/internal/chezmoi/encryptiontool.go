package chezmoi

// An EncryptionTool encrypts and decrypts files and data.
type EncryptionTool interface {
	Decrypt(ciphertext []byte) ([]byte, error)
	DecryptToFile(filename string, ciphertext []byte) error
	Encrypt(plaintext []byte) ([]byte, error)
	EncryptFile(filename string) ([]byte, error)
}
