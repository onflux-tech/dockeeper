package proxies

import (
	"net/url"
	"strings"
)

var sensitiveQueryKeys = map[string]bool{
	"token":        true,
	"key":          true,
	"password":     true,
	"secret":       true,
	"access_token": true,
	"api_key":      true,
	"apikey":       true,
}

func redactUpstream(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}

	if !strings.Contains(raw, "://") {
		if !strings.Contains(raw, "@") {
			return raw
		}
		parsed, err := url.Parse("http://" + raw)
		if err != nil {
			return raw
		}
		parsed.User = nil
		return strings.TrimPrefix(parsed.String(), "http://")
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	parsed.User = nil
	query := parsed.Query()
	for key := range query {
		if sensitiveQueryKeys[strings.ToLower(key)] {
			query.Set(key, "redacted")
		}
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}
