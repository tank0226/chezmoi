package chezmoi

import (
	"bytes"
	"os/exec"

	"github.com/rs/zerolog/log"

	"github.com/twpayne/chezmoi/chezmoi2/internal/chezmoilog"
)

// A GPGEncryption uses gpg for encryption and decryption. See https://gnupg.org/.
type GPGEncryption struct {
	Command   string
	Args      []string
	Recipient string
	Symmetric bool
}

// Decrypt implements Encyrption.Decrypt.
func (t *GPGEncryption) Decrypt(ciphertext []byte) ([]byte, error) {
	args := append([]string{"--decrypt"}, t.Args...)
	//nolint:gosec
	cmd := exec.Command(t.Command, args...)
	cmd.Stdin = bytes.NewReader(ciphertext)
	return chezmoilog.LogCmdOutput(log.Logger, cmd)
}

// DecryptToFile implements Encryption.DecryptToFile.
func (t *GPGEncryption) DecryptToFile(filename string, ciphertext []byte) error {
	args := append(append([]string{
		// "--batch",
		"--decrypt",
		"--output", filename,
		"--yes",
	}), t.Args...)
	//nolint:gosec
	return chezmoilog.LogCmdRun(log.Logger, exec.Command(t.Command, args...))
}

// Encrypt implements Encryption.Encrypt.
func (t *GPGEncryption) Encrypt(plaintext []byte) ([]byte, error) {
	args := append(t.encryptArgs(), t.Args...)
	//nolint:gosec
	return chezmoilog.LogCmdOutput(log.Logger, exec.Command(t.Command, args...))
}

// EncryptFile implements Encryption.EncryptFile.
func (t *GPGEncryption) EncryptFile(filename string) (ciphertext []byte, err error) {
	args := append(append(t.encryptArgs(),
		// "--batch",
		"--output", filename,
		"--yes",
	), t.Args...)
	//nolint:gosec
	return chezmoilog.LogCmdOutput(log.Logger, exec.Command(t.Command, args...))
}

func (t *GPGEncryption) encryptArgs() []string {
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
