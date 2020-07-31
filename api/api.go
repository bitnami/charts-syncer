//go:generate protoc -I=. --go_out=. config.proto

// Package api provides APIs for syncing a chart repository
package api

// SetBasicAuth is used configure the auth credentials for Repo kinds
func (r *Repo) SetBasicAuth(username string, password string) error {
	// No auth provided, leave as-is
	if username == "" && password == "" {
		return nil
	}
	// If username or password are already set, the value from
	// config file has preference over environment variable
	if r.Auth == nil {
		r.Auth = &Auth{Username: username, Password: password}
	} else {
		if r.Auth.Username == "" {
			r.Auth.Username = username
		}
		if r.Auth.Password == "" {
			r.Auth.Password = password
		}
	}
	return nil
}
