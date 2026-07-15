package image

import (
	"errors"
	"strings"

	"github.com/operator-framework/operator-manifest-tools/pkg/imageresolver"
)

// Resolve takes a list of images and returns a mapping of the images to an image name with a digest.
func Resolve(resolver imageresolver.ImageResolver, references []string) (Replacements, error) {
	results := make(map[string]string, len(references))
	for _, ref := range references {
		if strings.Contains(ref, "@") {
			// Already uses a digest
			continue
		}

		shaRef, err := resolver.ResolveImageReference(ref)
		if err != nil {
			return nil, errors.New("error resolving image: " + err.Error())
		}

		results[ref] = shaRef
	}

	return NewReplacements(results)
}
