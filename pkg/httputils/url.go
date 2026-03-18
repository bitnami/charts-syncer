package httputils

import (
	"net/url"
	"regexp"
	"strings"
)

var checkSchemaRegexp = regexp.MustCompile(`^[^:]+://`)

func RemoveSchema(ref string) string {
	if !checkSchemaRegexp.MatchString(ref) {
		return ref
	}

	u, err := url.Parse(ref)
	if err != nil {
		return ref
	}

	if u.Scheme != "" {
		u.Scheme = ""
		return strings.TrimLeft(u.String(), "/")
	}

	return ref
}
