package chartrepotest

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

// tChartMuseumReal starts a real ChartMuseum service on an available port,
// running in a Docker container. Most tests should use a fake instead.
//
// The URL of the service and a cleanup func that should be called once the
// service is no longer needed are returned.
//
// Any errors while setting up or tearing down the service will be reported on
// the *testing.T instance.
//
// Since pulling an image and running a service is time consuming, any tests
// using this are skipped for `-test.short` runs. The test is also skipped if no
// `docker` binary can be located.
func tChartMuseumReal(t *testing.T, username, password string) (string, func()) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	// Start chartmuseum in Docker publishing the default port to an available
	// host port.
	// Charts are stored in "local" storage backed by a Docker volume so they
	// can be deleted easily. The image runs as user 1000:1000 by default but is
	// overridden back to 0:0 here to avoid setting up volume permissions.
	chartsVolume := tDockerVolumeCreate(t)
	id := tDockerRunDetach(t,
		"--publish", "0:8080",
		"--volume", fmt.Sprintf("%s:/charts", chartsVolume),
		"--user", "0:0",
		"chartmuseum/chartmuseum",
		"--depth-dynamic", // Arbitrary multitenancy repo depth
		"--storage", "local", "--storage-local-rootdir", "/charts",
		"--basic-auth-user", username, "--basic-auth-pass", password,
	)
	port := strings.TrimSpace(tDockerExec(t,
		"inspect", "--format", `{{(index (index .NetworkSettings.Ports "8080/tcp") 0).HostPort}}`,
		id,
	))

	return "http://localhost:" + port, func() {
		tDockerExec(t, "stop", id)
		tDockerExec(t, "rm", id)
		tDockerVolumeRm(t, chartsVolume)
	}
}

func tDockerVolumeCreate(t *testing.T) string {
	return strings.TrimSpace(tDockerExec(t, "volume", "create"))
}

func tDockerVolumeRm(t *testing.T, volume string) string {
	return strings.TrimSpace(tDockerExec(t, "volume", "rm", volume))
}

func tDockerRunDetach(t *testing.T, args ...string) string {
	args = append([]string{"run", "-d"}, args...)
	return strings.TrimSpace(tDockerExec(t, args...))
}

func tDockerExec(t *testing.T, args ...string) string {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skipf("skipping test, `docker` not found: %s", err)
	}

	cmd := exec.Command("docker", args...)
	out, err := cmd.Output()
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			err = fmt.Errorf("%s: %w", exitError.Stderr, err)
		}
		t.Fatal(err)
	}
	return string(out)
}
