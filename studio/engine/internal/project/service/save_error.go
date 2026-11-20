package service

import (
	"errors"
	"io/fs"
	"os"
	"strings"

	"studio-engine/internal/apperr"
)

// saveFailed puts a failed Save in words the user can act on (fix plan A5). Before, a read-only,
// locked or full disk reached the page as the generic "internal error ... the server is
// overloaded". Windows' own reason is kept; the paths are left out.
func saveFailed(err error) error {
	reason := err.Error()
	var link *os.LinkError
	var path *fs.PathError
	switch {
	case errors.As(err, &link):
		reason = link.Err.Error()
	case errors.As(err, &path):
		reason = path.Err.Error()
	}
	return apperr.RuntimeError("ScadaBench could not write the project file (" + strings.TrimSuffix(reason, ".") +
		"). Check that the data folder is not read-only or full and that no other program has the project file " +
		"open, then press Save again.")
}
