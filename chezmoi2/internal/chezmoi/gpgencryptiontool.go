package chezmoi

import (
	"bytes"
	"os/exec"

	"github.com/rs/zerolog/log"

	"github.com/twpayne/chezmoi/chezmoi2/internal/chezmoilog"
)

// A GPGEncryptionTool uses gpg for encryption and decryption. See https://gnupg.org/.
type GPGEncryptionTool struct {
	Command   string
	Args      []string
	Recipient string
	Symmetric bool
}

// Decrypt implements EncyrptionTool.Decrypt.
func (t *GPGEncryptionTool) Decrypt(ciphertext []byte) ([]byte, error) {
	//nolint:gosec
	cmd := exec.Command(t.Command, append([]string{
		"--decrypt",
	}, t.Args...)...)
	cmd.Stdin = bytes.NewReader(ciphertext)
	return chezmoilog.LogCmdOutput(log.Logger, cmd)
}

// DecryptToFile implements EncryptionTool.DecryptToFile.
func (t *GPGEncryptionTool) DecryptToFile(filename string, ciphertext []byte) error {
	args := append([]string{"--decrypt", "--output", filename}, t.Args...)
	//nolint:gosec
	return chezmoilog.LogCmdRun(log.Logger, exec.Command(t.Command, args...))
}

// Encrypt implements EncryptionTool.Encrypt.
func (t *GPGEncryptionTool) Encrypt(plaintext []byte) ([]byte, error) {
	args := append(t.encyptArgs(), t.Args...)
	//nolint:gosec
	return chezmoilog.LogCmdOutput(log.Logger, exec.Command(t.Command, args...))
}

// EncryptFile implements EncryptionTool.EncryptFile.
func (t *GPGEncryptionTool) EncryptFile(filename string) (ciphertext []byte, err error) {
	args := append(append(t.encyptArgs(), "--output", filename), t.Args...)
	//nolint:gosec
	return chezmoilog.LogCmdOutput(log.Logger, exec.Command(t.Command, args...))
}

func (t *GPGEncryptionTool) encyptArgs() []string {
	args := []string{
		"--armor",
		"--encrypt",
	}
	if t.Recipient != "" {
		args = append(args, "--recipient", t.Recipient)
	}
	if t.Symmetric {
		args = append(args, "--symmetric")
	}
	return args
}
