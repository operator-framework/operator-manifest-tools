package imagename

import (
	"errors"
	"fmt"
	"strings"
)

type GetStringOptions []GetStringOption
type GetStringOption byte

const (
	Registry GetStringOption = 1 << iota
	Tag
	ExplicitTag
	ExplicitNamespace
	maxKey
)

var (
	DefaultGetStringOptions GetStringOption = Registry.Set(Tag)

	ErrNoImageRepository = errors.New("No image repository specified")
)

func (bs GetStringOptions) Combine() GetStringOption {
	var result GetStringOption

	for i := range bs {
		result = result.Set(bs[i])
	}

	return result
}
func (b GetStringOption) Set(flag GetStringOption) GetStringOption    { return b | flag }
func (b GetStringOption) Clear(flag GetStringOption) GetStringOption  { return b &^ flag }
func (b GetStringOption) Toggle(flag GetStringOption) GetStringOption { return b ^ flag }
func (b GetStringOption) Has(flag GetStringOption) bool            { return b&flag != 0 }

type ImageName struct {
	Registry  string
	Namespace string
	Repo      string
	Tag       string
}

func (imageName *ImageName) HasDigest() bool {
	return strings.HasPrefix(imageName.Tag, "sha256:")
}

func (imageName *ImageName) GetRepo(options ...GetStringOption) string {
	result := imageName.Repo

	optionSet := GetStringOptions(options).Combine()

	if imageName.Namespace != "" {
		result = fmt.Sprintf("%s/%s", imageName.Namespace, result)
	}

	if optionSet.Has(ExplicitNamespace) {
		result = fmt.Sprintf("%s/%s", "library", result)
	}

	return result
}

func (imageName *ImageName) ToString(options ...GetStringOption) (string, error) {

	if imageName.Repo == "" {
		return "", ErrNoImageRepository
	}

	optionSet := GetStringOptions(options).Combine()

	var str strings.Builder

	if optionSet.Has(Registry) && imageName.Registry != "" {
		str.WriteString(imageName.Registry)
		str.WriteString("/")
	}

	str.WriteString(imageName.GetRepo(optionSet))

	if imageName.Tag != "" {
		if imageName.HasDigest() {
			str.WriteString("@")
		} else {
			str.WriteString(":")
		}

		str.WriteString(imageName.Tag)
	} else if optionSet.Has(ExplicitTag) {
		str.WriteString(":latest")
	}

	return str.String(), nil
}

func (imageName *ImageName) Enclose(organization string) {
	if imageName.Namespace == organization {
		return
	}

	repoParts := []string{imageName.Repo}

	if imageName.Namespace != "" {
		repoParts = append([]string{imageName.Namespace}, repoParts...)
	}

	imageName.Namespace = organization
	imageName.Repo = strings.Join(repoParts, "-")
}

func (imageName *ImageName) String() string {
	result, err := imageName.ToString(Registry, Tag)

	if err != nil {
		panic(err)
	}

	return result
}

func Parse(imageName string) *ImageName {
	result := &ImageName{}

	s := strings.SplitN(imageName, "/", 3)
	if len(s) == 2 {
		if strings.ContainsAny(s[0], ".:") {
			result.Registry = s[0]
		} else {
			result.Namespace = s[0]
		}
	} else if len(s) == 3 {
		result.Registry = s[0]
		result.Namespace = s[1]
	}

	result.Repo = s[len(s)-1]
	result.Tag = "latest"

	if strings.ContainsAny(result.Repo, "@:") {
		s = strings.SplitN(result.Repo, "@", 2)

		if len(s) != 2 {
			s = strings.SplitN(result.Repo, ":", 2)
		}

		if len(s) == 2 {
			result.Repo, result.Tag = s[0], s[1]
		}
	}
	return result
}
