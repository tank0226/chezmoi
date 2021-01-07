package chezmoi

import (
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/twpayne/chezmoi/chezmoi2/internal/chezmoitest"
)

func TestGPGEncryption(t *testing.T) {
	t.Skip() // FIXME
	command, err := exec.LookPath("gpg")
	if errors.Is(err, exec.ErrNotFound) {
		t.Skip("gpg not found in $PATH")
	}
	require.NoError(t, err)

	tempDir, err := ioutil.TempDir("", "chezmoi-test-GPGEncryption")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.RemoveAll(tempDir))
	}()

	key, err := chezmoitest.GPGGenerateKey(tempDir)
	require.NoError(t, err)

	gpgEncryption := &GPGEncryption{
		Command: command,
		Args: []string{
			"--homedir", tempDir,
			"--no-tty",
			"--passphrase", "chezmoi-test-passphrase",
			"--pinentry-mode", "loopback",
		},
		Recipient: key,
	}

	testEncryptionDecryptToFile(t, gpgEncryption)
	testEncryptionEncryptDecrypt(t, gpgEncryption)
	testEncryptionEncryptFile(t, gpgEncryption)
}
