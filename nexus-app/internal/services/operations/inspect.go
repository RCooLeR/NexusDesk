package operations

import (
	"bytes"
	"io"
	"os"
	"regexp"
	"strings"
)

var (
	envAssignmentPattern = regexp.MustCompile(`(?m)^(\s*(?:export\s+)?[A-Za-z_][A-Za-z0-9_]*(?:SECRET|TOKEN|KEY|PASSWORD|PASS|PWD|CREDENTIAL|PRIVATE)[A-Za-z0-9_]*\s*=\s*)(.+)$`)
	yamlSecretPattern    = regexp.MustCompile(`(?im)^(\s*[A-Za-z0-9_.-]*(?:secret|token|key|password|pass|pwd|credential|private)[A-Za-z0-9_.-]*\s*:\s*)(.+)$`)
	jsonSecretPattern    = regexp.MustCompile(`(?im)^(\s*"[A-Za-z0-9_.-]*(?:secret|token|key|password|pass|pwd|credential|private)[A-Za-z0-9_.-]*"\s*:\s*)("[^"]*"|[^,\r\n]+)(,?)$`)
	redactedSecretMarker = "[REDACTED]"
)

func readBounded(path string, limit int64) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var buffer bytes.Buffer
	_, err = io.CopyN(&buffer, file, limit+1)
	if err != nil && err != io.EOF {
		return nil, err
	}
	content := buffer.Bytes()
	if int64(len(content)) > limit {
		content = content[:limit]
	}
	return content, nil
}

func redactSecrets(text string) string {
	text = envAssignmentPattern.ReplaceAllString(text, `${1}`+redactedSecretMarker)
	text = yamlSecretPattern.ReplaceAllString(text, `${1}`+redactedSecretMarker)
	text = jsonSecretPattern.ReplaceAllString(text, `${1}"`+redactedSecretMarker+`"${3}`)
	return strings.TrimRight(text, "\x00")
}
