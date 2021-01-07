package chezmoi

import (
	"bytes"
	"os/exec"

	"github.com/rs/zerolog/log"

	"github.com/twpayne/chezmoi/chezmoi2/internal/chezmoilog"
)

// An AGEEncryptionTool uses age for encryption and decryption. See
// https://github.com/FiloSottile/age.
type AGEEncryptionTool struct {
	Command    string
	Args       []string // FIXME
	Identity   string
	Identities []string
	Recipient  string
	Recipients []string
}

// Decrypt implements EncyrptionTool.Decrypt.
func (t *AGEEncryptionTool) Decrypt(ciphertext []byte) ([]byte, error) {
	//nolint:gosec
	cmd := exec.Command(t.Command, append(t.decryptArgs(), t.Args...)...)
	cmd.Stdin = bytes.NewReader(ciphertext)
	plaintext, err := chezmoilog.LogCmdOutput(log.Logger, cmd)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

// DecryptToFile implements EncryptionTool.DecryptToFile.
func (t *AGEEncryptionTool) DecryptToFile(filename string, ciphertext []byte) error {
	//nolint:gosec
	cmd := exec.Command(t.Command, append(append(t.decryptArgs(), "--output", filename), t.Args...)...)
	cmd.Stdin = bytes.NewReader(ciphertext)
	return chezmoilog.LogCmdRun(log.Logger, cmd)
}

// Encrypt implements EncryptionTool.Encrypt.
func (t *AGEEncryptionTool) Encrypt(plaintext []byte) ([]byte, error) {
	//nolint:gosec
	cmd := exec.Command(t.Command, append(t.encryptArgs(), t.Args...)...)
	cmd.Stdin = bytes.NewReader(plaintext)
	ciphertext, err := chezmoilog.LogCmdOutput(log.Logger, cmd)
	if err != nil {
		return nil, err
	}
	return ciphertext, nil
}

// EncryptFile implements EncryptionTool.EncryptFile.
func (t *AGEEncryptionTool) EncryptFile(filename string) ([]byte, error) {
	//nolint:gosec
	cmd := exec.Command(t.Command, append(append(t.encryptArgs(), t.Args...), filename)...)
	return chezmoilog.LogCmdOutput(log.Logger, cmd)
}

func (t *AGEEncryptionTool) decryptArgs() []string {
	args := make([]string, 0, 1+2*(1+len(t.Identities)))
	args = append(args, "--decrypt")
	if t.Identity != "" {
		args = append(args, "--identity", t.Identity)
	}
	for _, identity := range t.Identities {
		args = append(args, "--identity", identity)
	}
	return args
}

func (t *AGEEncryptionTool) encryptArgs() []string {
	args := make([]string, 0, 1+2*(1+len(t.Recipients)))
	args = append(args, "--armor")
	if t.Recipient != "" {
		args = append(args, "--recipient", t.Recipient)
	}
	for _, recipient := range t.Recipients {
		args = append(args, "--recipient", recipient)
	}
	return args
}
