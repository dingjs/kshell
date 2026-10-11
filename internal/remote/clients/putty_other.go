//go:build !windows

package clients

// PuTTYScanner 在非 Windows 上无注册表可读，Scan 返回空。
type PuTTYScanner struct{}

func (PuTTYScanner) ID() string { return "putty" }

func (PuTTYScanner) Scan() ([]ClientCandidate, error) {
	return nil, nil
}
