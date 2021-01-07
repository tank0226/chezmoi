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

var _ EncryptionTool = &nullEncryptionTool{}

type testEncryptionTool struct {
	key byte
}

var _ EncryptionTool = &testEncryptionTool{}

func newTestEncryptionTool() *testEncryptionTool {
	return &testEncryptionTool{
		key: byte(rand.Int() + 1),
	}
}

func (t *testEncryptionTool) Decrypt(ciphertext []byte) ([]byte, error) {
	return t.xorWithKey(ciphertext), nil
}

func (t *testEncryptionTool) DecryptToFile(filename string, ciphertext []byte) error {
	return ioutil.WriteFile(filename, t.xorWithKey(ciphertext), 0o666)
}

func (t *testEncryptionTool) Encrypt(plaintext []byte) ([]byte, error) {
	return t.xorWithKey(plaintext), nil
}

func (t *testEncryptionTool) EncryptFile(filename string) ([]byte, error) {
	plaintext, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return t.xorWithKey(plaintext), nil
}

func (t *testEncryptionTool) xorWithKey(input []byte) []byte {
	output := make([]byte, 0, len(input))
	for _, b := range input {
		output = append(output, b^t.key)
	}
	return output
}

func testEncryptionToolDecryptToFile(t *testing.T, et EncryptionTool) {
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

func testEncryptionToolEncryptDecrypt(t *testing.T, et EncryptionTool) {
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

func testEncryptionToolEncryptFile(t *testing.T, et EncryptionTool) {
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

func TestTestEncryptionTool(t *testing.T) {
	t.Helper()
	et := newTestEncryptionTool()
	testEncryptionToolDecryptToFile(t, et)
	testEncryptionToolEncryptDecrypt(t, et)
	testEncryptionToolEncryptFile(t, et)
}
