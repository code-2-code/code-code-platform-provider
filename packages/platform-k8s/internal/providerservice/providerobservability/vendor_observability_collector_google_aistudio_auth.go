package providerobservability

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

func googleAIStudioAuthorizationHeader(cookieHeader, origin string, now time.Time) (string, error) {
	return googleAIStudioDerivedAuthorizationHeader(cookieHeader, origin, now)
}

func googleAIStudioRequestAuthorizationHeader(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "Authorization:")
	value = strings.TrimPrefix(value, "authorization:")
	return strings.TrimSpace(value)
}

func googleAIStudioRequestAuthUser(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return googleAIStudioDefaultAuthUser
	}
	return value
}

func googleAIStudioDerivedAuthorizationHeader(cookieHeader, origin string, now time.Time) (string, error) {
	if strings.TrimSpace(origin) == "" {
		origin = googleAIStudioDefaultOrigin
	}
	timestamp := now.Unix()
	tokens := make([]string, 0, 3)
	if token := googleAIStudioAuthorizationToken(cookieHeader, origin, timestamp, "SAPISIDHASH", "SAPISID", "__Secure-3PAPISID"); token != "" {
		tokens = append(tokens, token)
	}
	if token := googleAIStudioAuthorizationToken(cookieHeader, origin, timestamp, "SAPISID1PHASH", "__Secure-1PAPISID"); token != "" {
		tokens = append(tokens, token)
	}
	if token := googleAIStudioAuthorizationToken(cookieHeader, origin, timestamp, "SAPISID3PHASH", "__Secure-3PAPISID"); token != "" {
		tokens = append(tokens, token)
	}
	if len(tokens) == 0 {
		return "", fmt.Errorf("google ai studio quotas: SAPISID, __Secure-1PAPISID, or __Secure-3PAPISID cookie is missing")
	}
	return strings.Join(tokens, " "), nil
}

func googleAIStudioAuthorizationToken(cookieHeader, origin string, timestamp int64, prefix string, cookieNames ...string) string {
	var cookieValue string
	for _, cookieName := range cookieNames {
		cookieValue = googleAIStudioCookieValue(cookieHeader, cookieName)
		if cookieValue != "" {
			break
		}
	}
	if cookieValue == "" {
		return ""
	}
	plain := fmt.Sprintf("%d %s %s", timestamp, cookieValue, strings.TrimSpace(origin))
	digest := sha1.Sum([]byte(plain))
	return fmt.Sprintf("%s %d_%s", prefix, timestamp, hex.EncodeToString(digest[:]))
}

func googleAIStudioCookieValue(cookieHeader, name string) string {
	for _, part := range strings.Split(cookieHeader, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key, value, ok := strings.Cut(part, "=")
		if !ok || strings.TrimSpace(key) != name {
			continue
		}
		return strings.TrimSpace(value)
	}
	return ""
}
