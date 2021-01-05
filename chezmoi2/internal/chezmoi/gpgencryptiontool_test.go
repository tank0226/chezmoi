package chezmoi

import (
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/twpayne/chezmoi/chezmoi2/internal/chezmoitest"
)

var gpgKeyMarkedAsUltimatelyTrustedRx = regexp.MustCompile(`gpg: key ([0-9A-F]+) marked as ultimately trusted`)

func TestGPGEncryptionTool(t *testing.T) {
	command, err := exec.LookPath("gpg")
	if errors.Is(err, exec.ErrNotFound) {
		t.Skip("gpg not found in $PATH")
	}
	require.NoError(t, err)

	tempDir, err := ioutil.TempDir("", "chezmoi-test-GPGEncryptionTool")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.RemoveAll(tempDir))
	}()

	key, err := chezmoitest.GPGQuickGenerateKey(tempDir)
	require.NoError(t, err)

	gpgEncryptionTool := &GPGEncryptionTool{
		Command: command,
		Args: []string{
			"--homedir", tempDir,
			"--no-tty",
			"--passphrase", "chezmoi-test-passphrase",
			"--pinentry-mode", "loopback",
		},
		Recipient: key,
	}
	testEncryptionToolDecryptToFile(t, gpgEncryptionTool)
	testEncryptionToolEncryptDecrypt(t, gpgEncryptionTool)
	testEncryptionToolEncryptFile(t, gpgEncryptionTool)
}
