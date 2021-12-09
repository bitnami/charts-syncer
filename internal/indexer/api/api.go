//go:generate prototool generate

// Package api provides APIs to index charts
package api

// Has returns whether the index has the requested chart name and version
func (x *Index) Has(name, version string) bool {
	for _, c := range x.GetCharts() {
		if c.GetName() != name {
			continue
		}
		for _, v := range c.GetVersions() {
			if v.GetVersion() == version {
				return true
			}
		}
	}
	return false
}
