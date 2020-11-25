package syncer

import (
	"os"
	"path"

	homedir "github.com/mitchellh/go-homedir"
)

// WorkdirName is the default name for a workdir
const WorkdirName = ".charts-syncer"

// DefaultWorkdir returns the default workdir path
func DefaultWorkdir() string {
	// We are ignoring errors here as they don't really matter for the purpose
	// of the function

	// Try to assign home as workdir
	home, _ := homedir.Dir()
	if home != "" {
		return path.Join(home, WorkdirName)
	}

	// Try to assign the current directory as workdir
	cwd, _ := os.Getwd()
	if cwd != "" {
		return path.Join(cwd, WorkdirName)
	}

	return WorkdirName
}
