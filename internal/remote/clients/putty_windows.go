//go:build windows

package clients

import (
	"strconv"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// PuTTYScanner 从注册表读取 PuTTY 已保存的 SSH 会话。
type PuTTYScanner struct{}

func (PuTTYScanner) ID() string { return "putty" }

const puttySessionsKey = `Software\SimonTatham\PuTTY\Sessions`

func (PuTTYScanner) Scan() ([]ClientCandidate, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, puttySessionsKey, registry.READ)
	if err != nil {
		return nil, nil
	}
	defer k.Close()

	names, err := k.ReadSubKeyNames(-1)
	if err != nil {
		return nil, nil
	}

	var out []ClientCandidate
	for _, name := range names {
		if name == "" || name == "Default%20Settings" {
			continue
		}
		sessionKey, err := registry.OpenKey(k, name, registry.READ)
		if err != nil {
			continue
		}
		fields, err := readPuTTYRegistryValues(sessionKey)
		sessionKey.Close()
		if err != nil {
			continue
		}
		displayName := decodePuttyRegName(name)
		cand, ok := parsePuTTYSession(displayName, fields)
		if ok {
			out = append(out, cand)
		}
	}
	return out, nil
}

func readPuTTYRegistryValues(k registry.Key) (puTTYFieldValues, error) {
	names, err := k.ReadValueNames(-1)
	if err != nil {
		return nil, err
	}
	out := puTTYFieldValues{}
	for _, n := range names {
		val, _, err := k.GetStringValue(n)
		if err != nil {
			if iv, _, err2 := k.GetIntegerValue(n); err2 == nil {
				out[n] = strconv.FormatUint(iv, 10)
				continue
			}
			continue
		}
		out[n] = val
	}
	return out, nil
}

func decodePuttyRegName(name string) string {
	replacer := strings.NewReplacer("%20", " ", "%25", "%")
	return replacer.Replace(name)
}
