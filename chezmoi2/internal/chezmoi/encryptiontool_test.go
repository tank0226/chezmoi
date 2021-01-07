package chezmoi

import (
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ Encryption = &nullEncryption{}

type testEncryption struct {
	key byte
}

var _ Encryption = &testEncryption{}

func newTestEncryption() *testEncryption {
	return &testEncryption{
		key: byte(rand.Int() + 1),
	}
}

func (t *testEncryption) Decrypt(ciphertext []byte) ([]byte, error) {
	return t.xorWithKey(ciphertext), nil
}

func (t *testEncryption) DecryptToFile(filename string, ciphertext []byte) error {
	return ioutil.WriteFile(filename, t.xorWithKey(ciphertext), 0o666)
}

func (t *testEncryption) Encrypt(plaintext []byte) ([]byte, error) {
	return t.xorWithKey(plaintext), nil
}

func (t *testEncryption) EncryptFile(filename string) ([]byte, error) {
	plaintext, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return t.xorWithKey(plaintext), nil
}

func (t *testEncryption) xorWithKey(input []byte) []byte {
	output := make([]byte, 0, len(input))
	for _, b := range input {
		output = append(output, b^t.key)
	}
	return output
}

func testEncryptionDecryptToFile(t *testing.T, et Encryption) {
	t.Helper()
	t.Run("DecryptToFile", func(t *testing.T) {
		expectedPlaintext := []byte("plaintext")

		actualCiphertext, err := et.Encrypt(expectedPlaintext)
		require.NoError(t, err)
		assert.NotEqual(t, expectedPlaintext, actualCiphertext)

		tempDir, err := ioutil.TempDir("", "chezmoi-test-encryption-tool")
		require.NoError(t, err)
		defer func() {
			assert.NoError(t, os.RemoveAll(tempDir))
		}()
		filename := filepath.Join(tempDir, "filename")

		require.NoError(t, et.DecryptToFile(filename, actualCiphertext))

		actualPlaintext, err := ioutil.ReadFile(filename)
		require.NoError(t, err)
		assert.Equal(t, expectedPlaintext, actualPlaintext)
	})
}

func testEncryptionEncryptDecrypt(t *testing.T, et Encryption) {
	t.Helper()
	t.Run("EncryptDecrypt", func(t *testing.T) {
		expectedPlaintext := []byte("plaintext")

		actualCiphertext, err := et.Encrypt(expectedPlaintext)
		require.NoError(t, err)
		assert.NotEqual(t, expectedPlaintext, actualCiphertext)

		actualPlaintext, err := et.Decrypt(actualCiphertext)
		require.NoError(t, err)
		assert.Equal(t, expectedPlaintext, actualPlaintext)
	})
}

func testEncryptionEncryptFile(t *testing.T, et Encryption) {
	t.Helper()
	t.Run("EncryptFile", func(t *testing.T) {
		expectedPlaintext := []byte("plaintext")

		tempDir, err := ioutil.TempDir("", "chezmoi-test-encryption-tool")
		require.NoError(t, err)
		defer func() {
			assert.NoError(t, os.RemoveAll(tempDir))
		}()
		filename := filepath.Join(tempDir, "filename")
		require.NoError(t, ioutil.WriteFile(filename, expectedPlaintext, 0o666))

		actualCiphertext, err := et.EncryptFile(filename)
		require.NoError(t, err)
		assert.NotEqual(t, expectedPlaintext, actualCiphertext)

		actualPlaintext, err := et.Decrypt(actualCiphertext)
		require.NoError(t, err)
		assert.Equal(t, expectedPlaintext, actualPlaintext)
	})
}

func TestTestEncryption(t *testing.T) {
	t.Helper()
	et := newTestEncryption()
	testEncryptionDecryptToFile(t, et)
	testEncryptionEncryptDecrypt(t, et)
	testEncryptionEncryptFile(t, et)
}
