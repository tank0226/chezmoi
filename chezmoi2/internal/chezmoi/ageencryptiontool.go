package chezmoi

import (
	"bytes"
	"os/exec"
)

// An AGEEncryptionTool uses age for encryption and decryption. See
// https://github.com/FiloSottile/age.
type AGEEncryptionTool struct {
	Command    string
	Args       []string
	Identity   string
	Identities []string
	Recipient  string
	Recipients []string
}

// Decrypt implements EncyrptionTool.Decrypt.
func (t *AGEEncryptionTool) Decrypt(filenameHint string, ciphertext []byte) ([]byte, error) {
	cmd := exec.Command(t.Command, t.encryptArgs()...)
	cmd.Stdin = bytes.NewReader(ciphertext)
	output := &bytes.Buffer{}
	cmd.Stdout = output
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

// DecryptToFile implements EncryptionTool.DecryptToFile.
func (t *AGEEncryptionTool) DecryptToFile(filenameHint string, ciphertext []byte) (string, func() error, error) {
	return "", func() error { return nil }, nil // FIXME IAMHERE
	// FIXME change EncryptionTool interface to pass filename, not filenameHint
}

// Encrypt implements EncryptionTool.Encrypt.
func (t *AGEEncryptionTool) Encrypt(plaintext []byte) ([]byte, error) {
	cmd := exec.Command(t.Command, t.encryptArgs()...)
	cmd.Stdin = bytes.NewReader(plaintext)
	output := &bytes.Buffer{}
	cmd.Stdout = output
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

// EncryptFile implements EncryptionTool.EncryptFile.
func (t *AGEEncryptionTool) EncryptFile(filename string) ([]byte, error) {
	return exec.Command(t.Command, append(t.encryptArgs(), filename)...).Output()
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
