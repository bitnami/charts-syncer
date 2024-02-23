package main

import (
	"bytes"
	"flag"
	"os"
	"os/exec"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	if os.Getenv("BE_CHARTSYNCER") == "1" {
		main()
		os.Exit(0)
		return
	}
	flag.Parse()
	c := m.Run()
	os.Exit(c)
}

// chartsyncer calls the chartsyncer command externally via exec
func chartsyncer(cmdArgs ...string) CmdResult {
	return execCommand(cmdArgs...)
}

func execCommand(args ...string) CmdResult {
	var buffStdout, buffStderr bytes.Buffer
	code := 0

	cmd := exec.Command(os.Args[0], args...)
	cmd.Stdout = &buffStdout
	cmd.Stderr = &buffStderr

	cmd.Env = append(os.Environ(), "BE_CHARTSYNCER=1")

	err := cmd.Run()

	if err != nil {
		code = err.(*exec.ExitError).Sys().(syscall.WaitStatus).ExitStatus()
	}

	return CmdResult{code: code, stdout: buffStdout.String(), stderr: buffStderr.String()}
}

type CmdResult struct {
	code   int
	stdout string
	stderr string
}

func (r CmdResult) AssertErrorMatch(t *testing.T, re interface{}) bool {
	if r.AssertError(t) {
		return assert.Regexp(t, re, r.stderr)
	}
	return true
}

func (r CmdResult) AssertSuccessMatchStdout(t *testing.T, re interface{}) bool {
	if r.AssertSuccess(t) {
		return assert.Regexp(t, re, r.stdout)
	}
	return true
}

func (r CmdResult) AssertSuccessMatchStderr(t *testing.T, re interface{}) bool {
	if r.AssertSuccess(t) {
		return assert.Regexp(t, re, r.stderr)
	}
	return true
}

func (r CmdResult) AssertCode(t *testing.T, code int) bool {
	return assert.Equal(t, code, r.code, "Expected %d code but got %d", code, r.code)
}

func (r CmdResult) AssertSuccess(t *testing.T) bool {
	return assert.True(t, r.Success(), "Expected command to success but got code=%d stderr=%s", r.code, r.stderr)
}

func (r CmdResult) AssertError(t *testing.T) bool {
	return assert.False(t, r.Success(), "Expected command to fail")
}

func (r CmdResult) Success() bool {
	return r.code == 0
}
