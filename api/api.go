//go:generate protoc -I=. --go_out=. config.proto

// Package api provides APIs for syncing a chart repository
package api

// SetBasicAuth is used configure the auth credentials for Repo kinds
func (r *Repo) SetBasicAuth(username string, password string) error {
	// No auth provided, leave as-is
	if username == "" && password == "" {
		return nil
	}
	// If username or password are already set, the value will
	// be override with the environment variable one as it
	// has preference.
	if r.Auth == nil {
		r.Auth = &Auth{Username: username, Password: password}
	} else {
		if username != "" {
			r.Auth.Username = username
		}
		if password != "" {
			r.Auth.Password = password
		}
	}
	return nil
}
