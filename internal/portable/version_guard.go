package portable

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/zax0rz/darkpawns/internal/releaseauth"
)

// compareReleaseVersions orders two stable release tags (vMAJOR.MINOR.PATCH).
// It returns -1 when a is older than b, 0 when equal, 1 when newer. A tag
// that is not a strict stable release is not comparable: the caller cannot
// judge direction and must not guess (pre-release and rehearsal builds flow
// through --archive, and a wrong guess here would strand an operator).
func compareReleaseVersions(a, b string) (int, error) {
	nums := func(tag string) ([3]int, error) {
		var out [3]int
		if !releaseauth.StableTag.MatchString(tag) {
			return out, fmt.Errorf("%q is not a stable release tag", tag)
		}
		for i, part := range strings.Split(strings.TrimPrefix(tag, "v"), ".") {
			n, err := strconv.Atoi(part)
			if err != nil {
				return out, err
			}
			out[i] = n
		}
		return out, nil
	}
	an, err := nums(a)
	if err != nil {
		return 0, err
	}
	bn, err := nums(b)
	if err != nil {
		return 0, err
	}
	for i := range an {
		switch {
		case an[i] < bn[i]:
			return -1, nil
		case an[i] > bn[i]:
			return 1, nil
		}
	}
	return 0, nil
}

// versionChangeDecision refuses a candidate that would move the instance
// backwards: an older release, or the same version built from a different
// commit. Downgrades lose migrations and data-format handling the running
// release already understands, and an equal-version swap can silently change
// behavior while claiming not to. --allow-downgrade (the console never
// passes it) takes responsibility for both. Equal version and commit is the
// updater's own "already up to date" case and is not a refusal.
func versionChangeDecision(current, target Manifest, allowDowngrade bool) error {
	if allowDowngrade {
		return nil
	}
	if current.Version == target.Version {
		if current.Commit != target.Commit {
			return fmt.Errorf("%s is already running with a different build (%s vs %s); your world was not stopped; rerun with --allow-downgrade to force", current.Version, current.Commit, target.Commit)
		}
		return nil
	}
	direction, err := compareReleaseVersions(current.Version, target.Version)
	if err != nil {
		// Not comparable (a rehearsal or archive build): the guard cannot
		// judge direction, so installing is the operator's explicit choice.
		return nil
	}
	if direction > 0 {
		return fmt.Errorf("refusing to downgrade from %s to %s; your world was not stopped; rerun with --allow-downgrade to force", current.Version, target.Version)
	}
	return nil
}
