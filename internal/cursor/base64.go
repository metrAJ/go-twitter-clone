package cursor

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"
)

type Base64Codec struct{}

func NewBase64Codec() *Base64Codec {
	return &Base64Codec{}
}

func (c *Base64Codec) Encode(t time.Time, id string) string {
	raw := fmt.Sprintf("%s|%s", t.Format(time.RFC3339Nano), id)
	return base64.StdEncoding.EncodeToString([]byte(raw))
}

func (c *Base64Codec) Decode(encoded string) (time.Time, string, error) {
	decodedBytes, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("failed to decode base64: %w", err)
	}

	parts := strings.Split(string(decodedBytes), "|")
	if len(parts) != 2 {
		return time.Time{}, "", fmt.Errorf("invalid cursor format")
	}

	t, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, "", err
	}

	return t, parts[1], nil
}
