package clients

import (
	"bufio"
	"encoding/xml"
	"io"
	"os"
	"strconv"
	"strings"
)

// MobaXtermScanner 扫描 MobaXterm.ini 或 .mxtsessions。
type MobaXtermScanner struct {
	// ConfigPaths 可注入；为空时使用常见默认路径。
	ConfigPaths []string
}

func (MobaXtermScanner) ID() string { return "mobaxterm" }

func (s MobaXtermScanner) Scan() ([]ClientCandidate, error) {
	if !platformWindows() {
		return nil, nil
	}
	paths := s.ConfigPaths
	if len(paths) == 0 {
		paths = defaultMobaXtermConfigPaths()
	}

	var out []ClientCandidate
	seen := map[string]bool{}
	for _, path := range paths {
		cands, err := parseMobaXtermFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		for _, c := range cands {
			k := strings.ToLower(c.Host) + "|" + strings.ToLower(c.User) + "|" + strconv.Itoa(c.Port)
			if seen[k] {
				continue
			}
			seen[k] = true
			out = append(out, c)
		}
	}
	return out, nil
}

func parseMobaXtermFile(path string) ([]ClientCandidate, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	text := string(data)
	if strings.Contains(text, "<MobaXterm") {
		if xmlCands := parseMobaXtermXML(data, path); len(xmlCands) > 0 {
			return xmlCands, nil
		}
	}
	return parseMobaXtermINI(text, path), nil
}

func parseMobaXtermINI(text, sourceFile string) []ClientCandidate {
	var out []ClientCandidate
	sc := bufio.NewScanner(strings.NewReader(text))
	var section string
	var sectKeys map[string]string

	flushSection := func() {
		if sectKeys == nil {
			return
		}
		host := firstNonEmpty(sectKeys, "host", "hostname")
		if host != "" {
			name := firstNonEmpty(sectKeys, "session", "session name", "name")
			out = append(out, mobaCandidateFromFields(sourceFile, name, host,
				firstNonEmpty(sectKeys, "username", "user"), sectKeys["port"]))
		}
		sectKeys = nil
	}

	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			flushSection()
			section = strings.ToLower(strings.Trim(line, "[]"))
			if strings.Contains(section, "session") && !strings.Contains(section, "bookmark") {
				sectKeys = map[string]string{}
			}
			continue
		}

		if sectKeys != nil {
			key, val, ok := splitINIAssignment(line)
			if ok {
				sectKeys[strings.ToLower(key)] = val
			}
			continue
		}

		lowSect := section
		if strings.Contains(lowSect, "bookmark") || lowSect == "saved sessions" {
			if idx := strings.Index(line, "="); idx > 0 {
				val := strings.TrimSpace(line[idx+1:])
				host, user, port, ok := mobaSessionFromBookmarkValue(val)
				if !ok {
					continue
				}
				name := strings.TrimSpace(line[:idx])
				out = append(out, ClientCandidate{
					Name:       name,
					Host:       host,
					User:       user,
					Port:       port,
					Source:     "mobaxterm",
					SourceFile: sourceFile,
					Confidence: "medium",
				})
			}
		}
	}
	flushSection()
	return out
}

func mobaCandidateFromFields(sourceFile, name, host, user, portStr string) ClientCandidate {
	port := 22
	if portStr != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(portStr)); err == nil && n > 0 {
			port = n
		}
	}
	if name == "" {
		name = host
	}
	return ClientCandidate{
		Name:       name,
		Host:       host,
		User:       user,
		Port:       port,
		Source:     "mobaxterm",
		SourceFile: sourceFile,
		Confidence: "medium",
	}
}

type mobaXMLRoot struct {
	XMLName  xml.Name      `xml:"MobaXterm"`
	Sessions []mobaSession `xml:"Sessions>Session"`
}

type mobaSession struct {
	Name     string `xml:"Name,attr"`
	Host     string `xml:"Host,attr"`
	Username string `xml:"Username,attr"`
	Port     string `xml:"Port,attr"`
}

func parseMobaXtermXML(data []byte, sourceFile string) []ClientCandidate {
	var root mobaXMLRoot
	if err := xml.Unmarshal(data, &root); err != nil {
		return nil
	}
	var out []ClientCandidate
	for _, s := range root.Sessions {
		if s.Host == "" {
			continue
		}
		out = append(out, mobaCandidateFromFields(sourceFile, s.Name, s.Host, s.Username, s.Port))
	}
	return out
}
