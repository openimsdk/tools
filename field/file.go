package field

import (
	"errors"
	"os"
)

// LinkTreatment is the base type for constants used by Exists that indicate
// how symlinks are treated for existence checks.
type LinkTreatment int

const (
	// CheckFollowSymlink follows the symlink and verifies that the target of
	// the symlink exists.
	CheckFollowSymlink LinkTreatment = iota

	// CheckSymlinkOnly does not follow the symlink and verifies only that they
	// symlink itself exists.
	CheckSymlinkOnly
)

// ErrInvalidLinkTreatment indicates that the link treatment behavior requested
// is not a valid behavior.
var ErrInvalidLinkTreatment = errors.New("unknown link behavior")

// Exists checks if specified file, directory, or symlink exists. The behavior
// of the test depends on the linkBehaviour argument. See LinkTreatment for
// more details.
func Exists(linkBehavior LinkTreatment, filename string) (bool, error) {
	var err error

	if linkBehavior == CheckFollowSymlink {
		_, err = os.Stat(filename)
	} else if linkBehavior == CheckSymlinkOnly {
		_, err = os.Lstat(filename)
	} else {
		return false, ErrInvalidLinkTreatment
	}

	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

// ReadDirNoStat returns a string of files/directories contained
// in dirname without calling lstat on them.
func ReadDirNoStat(dirname string) ([]string, error) {
	if dirname == "" {
		dirname = "."
	}

	f, err := os.Open(dirname)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return f.Readdirnames(-1)
}
