//go:generate prototool generate

// Package api provides APIs to index charts
package api

// Has returns whether the index has the requested chart name and version
func (x *Index) Has(name, version string) bool {
	if x.Entries[name] != nil {
		for _, v := range x.Entries[name].GetVersions() {
			if v.GetVersion() == version {
				return true
			}
		}
	}
	return false
}
