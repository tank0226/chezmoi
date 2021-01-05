package chezmoi

import (
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
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

	output, err := exec.Command(
		"gpg",
		"--batch",
		"--homedir", tempDir,
		"--no-tty",
		"--passphrase", "chezmoi-test-passphrase",
		"--pinentry-mode", "loopback",
		"--quick-generate-key", "chezmoi-test-key",
	).CombinedOutput()
	require.NoError(t, err)
	submatch := gpgKeyMarkedAsUltimatelyTrustedRx.FindSubmatch(output)
	require.NotNil(t, submatch)
	key := submatch[1]

	gpgET := &GPGEncryptionTool{
		Command: command,
		Args: []string{
			"--homedir", tempDir,
			"--no-tty",
			"--passphrase", "chezmoi-test-passphrase",
			"--pinentry-mode", "loopback",
		},
		Recipient: string(key),
	}
	testEncryptionToolDecryptToFile(t, gpgET)
	testEncryptionToolEncryptDecrypt(t, gpgET)
	testEncryptionToolEncryptFile(t, gpgET)
}
