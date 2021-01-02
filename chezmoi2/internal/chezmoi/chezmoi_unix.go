// +build !windows

package chezmoi

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

	vfs "github.com/twpayne/go-vfs"
)

var (
	umask        os.FileMode
	whitespaceRx = regexp.MustCompile(`\s+`)
)

func init() {
	umask = os.FileMode(syscall.Umask(0))
	syscall.Umask(int(umask))
}

// NewAbsPath returns a new AbsPath.
func NewAbsPath(path string) (AbsPath, error) {
	if path == "" || path[0] != '/' {
		return "", fmt.Errorf("%s: not an absolute path", path)
	}
	return AbsPath(path), nil
}

// ExpandTilde expands a leading tilde in path.
func ExpandTilde(path string, homeDirAbsPath AbsPath) string {
	switch {
	case path == "~":
		return string(homeDirAbsPath)
	case strings.HasPrefix(path, "~/"):
		return string(homeDirAbsPath.Join(RelPath(path[2:])))
	default:
		return path
	}
}

// FQDNHostname returns the FQDN hostname.
func FQDNHostname(fs vfs.FS) (string, error) {
	if fqdnHostname, err := etcHostsFQDNHostname(fs); err == nil && fqdnHostname != "" {
		return fqdnHostname, nil
	}
	return lookupAddrFQDNHostname()
}

// GetUmask returns the umask.
func GetUmask() os.FileMode {
	return umask
}

// NormalizePath returns path normalized. On non-Windows systems, normalized
// paths are absolute paths.
func NormalizePath(path string) (AbsPath, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return AbsPath(absPath), nil
}

// SetUmask sets the umask.
func SetUmask(newUmask os.FileMode) {
	umask = newUmask
	syscall.Umask(int(umask))
}

// etcHostsFQDNHostname returns the FQDN hostname from parsing /etc/hosts.
func etcHostsFQDNHostname(fs vfs.FS) (string, error) {
	etcHostsContents, err := fs.ReadFile("/etc/hosts")
	if err != nil {
		return "", err
	}
	s := bufio.NewScanner(bytes.NewReader(etcHostsContents))
	for s.Scan() {
		text := s.Text()
		text = strings.TrimSpace(text)
		if index := strings.IndexByte(text, '#'); index != -1 {
			text = text[:index]
		}
		fields := whitespaceRx.Split(text, -1)
		if len(fields) >= 2 && fields[0] == "127.0.1.1" {
			return fields[1], nil
		}
	}
	return "", s.Err()
}

// isExecutable returns if info is executable.
func isExecutable(info os.FileInfo) bool {
	return info.Mode().Perm()&0o111 != 0
}

// isPrivate returns if info is private.
func isPrivate(info os.FileInfo) bool {
	return info.Mode().Perm()&0o77 == 0
}

// lookupAddrFQDNHostname returns the FQDN hostname by doing a reverse lookup of
// 127.0.1.1.
func lookupAddrFQDNHostname() (string, error) {
	names, err := net.LookupAddr("127.0.1.1")
	if err != nil {
		return "", err
	}
	if len(names) == 0 {
		return "", nil
	}
	return strings.TrimSuffix(names[0], "."), nil
}

// umaskPermEqual returns if two permissions are equal after applying umask.
func umaskPermEqual(perm1, perm2, umask os.FileMode) bool {
	return perm1&^umask == perm2&^umask
}
