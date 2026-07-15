package imageresolver

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// Script supports using a script/executable as an
// image resolver. The script only needs to return the digest.
// Examples of custom resolvers can be found in the hack/resolvers
// folder on the repo.
type Script struct {
	path string
}

// ResolveImageReference resolves an image reference to its digest form using a custom script.
func (custom *Script) ResolveImageReference(imageReference string) (string, error) {
	path, err := filepath.Abs(custom.path)
	if err != nil {
		return "", err
	}

	output, err := exec.Command(path, imageReference).CombinedOutput() //nolint:gosec // script path is user-provided CLI input
	if err != nil {
		return "", err
	}

	imageName, err := getName(imageReference)
	if err != nil {
		return "", err
	}

	digest := strings.TrimSpace(string(output))
	return fmt.Sprintf("%s@sha256:%s", imageName, digest), nil
}
