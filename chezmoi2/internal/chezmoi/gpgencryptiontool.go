package chezmoi

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path"

	"go.uber.org/multierr"
)

// A GPGEncryptionTool uses gpg for encryption and decryption.
type GPGEncryptionTool struct {
	Command   string
	Args      []string
	Recipient string
	Symmetric bool
}

// Decrypt implements EncyrptionTool.Decrypt.
func (t *GPGEncryptionTool) Decrypt(filenameHint string, ciphertext []byte) ([]byte, error) {
	return encryptionToolDecrypt(t, filenameHint, ciphertext)
}

// DecryptToFile implements EncryptionTool.DecryptToFile.
func (t *GPGEncryptionTool) DecryptToFile(filenameHint string, ciphertext []byte) (filename string, cleanupFunc func() error, err error) {
	tempDir, err := ioutil.TempDir("", "chezmoi-gpg-decrypt")
	if err != nil {
		return
	}
	cleanupFunc = func() error {
		return os.RemoveAll(tempDir)
	}

	filename = path.Join(tempDir, path.Base(filenameHint))
	inputFilename := filename + ".asc"
	if err = ioutil.WriteFile(inputFilename, ciphertext, 0o600); err != nil {
		err = multierr.Append(err, cleanupFunc())
		return
	}

	args := []string{
		"--decrypt",
		"--output", filename,
		"--quiet",
		inputFilename,
	}

	if err = t.runWithArgs(args); err != nil {
		err = multierr.Append(err, cleanupFunc())
		return
	}

	return
}

// Encrypt implements EncryptionTool.Encrypt.
func (t *GPGEncryptionTool) Encrypt(plaintext []byte) (ciphertext []byte, err error) {
	return encryptionToolEncrypt(t, "chezmoi-gpg-encrypt", plaintext)
}

// EncryptFile implements EncryptionTool.EncryptFile.
func (t *GPGEncryptionTool) EncryptFile(filename string) (ciphertext []byte, err error) {
	tempDir, err := ioutil.TempDir("", "chezmoi-gpg-encrypt")
	if err != nil {
		return
	}
	defer func() {
		err = multierr.Append(err, os.RemoveAll(tempDir))
	}()

	outputFilename := path.Join(tempDir, path.Base(filename)+".gpg")
	args := []string{
		"--armor",
		"--encrypt",
		"--output", outputFilename,
		"--quiet",
	}
	if t.Recipient != "" {
		args = append(args, "--recipient", t.Recipient)
	}
	if t.Symmetric {
		args = append(args, "--symmetric")
	}
	args = append(args, filename)

	if err = t.runWithArgs(args); err != nil {
		return
	}

	ciphertext, err = ioutil.ReadFile(outputFilename)
	return
}

func (t *GPGEncryptionTool) runWithArgs(args []string) error {
	//nolint:gosec
	cmd := exec.Command(t.Command, append(t.Args, args...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
