package containersyncer

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/Masterminds/semver/v3"
)

type parsedTag struct {
	raw string
	ver *semver.Version
	rev int
}

func getHighestVersionTag(tags []string) (string, error) {
	if len(tags) == 0 {
		return "", fmt.Errorf("empty tag list")
	}

	var pt []parsedTag
	// Regex to capture the revision number at the end (e.g., -r24)
	reRev := regexp.MustCompile(`-r(\d+)$`)

	for _, t := range tags {
		// 1. Isolate SemVer base (13.16.0) from the rest (-photon...)
		parts := strings.SplitN(t, "-", 2)
		v, err := semver.NewVersion(parts[0])
		if err != nil {
			continue // Skip invalid semver formats
		}

		// 2. Extract revision integer
		rev := 0
		if matches := reRev.FindStringSubmatch(t); len(matches) > 1 {
			rev, _ = strconv.Atoi(matches[1]) // Error ignored as regex ensures digits
		}

		pt = append(pt, parsedTag{raw: t, ver: v, rev: rev})
	}

	if len(pt) == 0 {
		return "", fmt.Errorf("no valid tags parsed")
	}

	// 3. Sort ascending: Primary key = Version, Secondary key = Revision
	sort.Slice(pt, func(i, j int) bool {
		if !pt[i].ver.Equal(pt[j].ver) {
			return pt[i].ver.LessThan(pt[j].ver)
		}
		return pt[i].rev < pt[j].rev
	})

	// Return the last element (highest value)
	return pt[len(pt)-1].raw, nil
}
