package chezmoi

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/twpayne/chezmoi/chezmoi2/internal/chezmoitest"
)

func TestAGEEncryptionTool(t *testing.T) {
	command, err := exec.LookPath("age")
	if errors.Is(err, exec.ErrNotFound) {
		t.Skip("age not found in $PATH")
	}
	require.NoError(t, err)

	publicKey, privateKeyFile, err := chezmoitest.AGEGenerateKey("")
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, os.RemoveAll(filepath.Dir(privateKeyFile)))
	}()

	ageEncryptionTool := &AGEEncryptionTool{
		Command:   command,
		Identity:  privateKeyFile,
		Recipient: publicKey,
	}

	testEncryptionToolDecryptToFile(t, ageEncryptionTool)
	testEncryptionToolEncryptDecrypt(t, ageEncryptionTool)
	testEncryptionToolEncryptFile(t, ageEncryptionTool)
}
