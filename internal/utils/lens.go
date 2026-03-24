package utils

import (
	"strconv"
	"strings"
)

// lensBuilder is used to construct lens
type lensBuilder struct {
	funcs []func(any) (any, error)
	path  []string
}

// lens holds a series of functions that helps navigate a map[string]any data structure
type lens struct {
	funcs []func(any) (any, error)
}

// newLens creates a new lens builder
func Lens() *lensBuilder {
	return &lensBuilder{
		funcs: []func(any) (any, error){},
		path:  []string{},
	}
}

// L will create a step on a lens to navigate a slice by integer
func (d *lensBuilder) L(i int) *lensBuilder {
	d.path = append(d.path, strconv.Itoa(i))
	d.funcs = append(d.funcs, func(data any) (any, error) {
		slice, ok := data.([]any)

		if !ok {
			return nil, NewError(ErrNotFound, "expected a []any type on step %s path %s", strconv.Itoa(i), strings.Join(d.path, ","))
		}

		if i < 0 || i >= len(slice) {
			return nil, NewError(ErrNotFound, "not found on step %s path %s", strconv.Itoa(i), strings.Join(d.path, ","))
		}

		return slice[i], nil
	})

	return d
}

// M will create a step on a lens to navigate a map by a key
func (d *lensBuilder) M(key string) *lensBuilder {
	d.path = append(d.path, key)
	d.funcs = append(d.funcs, func(data any) (any, error) {
		mmap, ok := data.(map[string]any)

		if !ok {
			return nil, NewError(ErrNotFound, "expected a map[string]any type on step %s path %s", key, strings.Join(d.path, ","))
		}

		v, ok := mmap[key]
		if !ok {
			return nil, NewError(ErrNotFound, "not found on step %s path %s", key, strings.Join(d.path, ","))
		}

		return v, nil
	})
	return d
}

// Apply will add a step on a lens to loop over a slice and apply another
// lens to each element.
func (d *lensBuilder) Apply(l lens) *lensBuilder {
	d.path = append(d.path, "*")
	d.funcs = append(d.funcs, func(data any) (any, error) {
		slice, ok := data.([]any)

		if !ok {
			return nil, NewError(ErrNotFound, "expected a []any type on step * path %s", strings.Join(d.path, ","))
		}

		results := make([]any, 0, len(slice))

		for i := range slice {
			data := slice[i]

			result, err := l.Lookup(data)

			if err != nil {
				continue
			}

			if result != nil {
				results = append(results, result)
			}
		}

		return results, nil
	})
	return d
}

// Build finalizes the lens steps and makes it able to return results.
func (d *lensBuilder) Build() lens {
	funcs := make([]func(any) (any, error), 0, len(d.funcs))
	funcs = append(funcs, d.funcs...)

	return lens{
		funcs: funcs,
	}
}

// Lookup will run the lens against data, returning a result or error.
func (l lens) Lookup(data any) (result any, err error) {
	result = data
	for _, fun := range l.funcs {
		result, err = fun(result)

		if err != nil {
			return
		}
	}

	return
}

// L is an alias for Lookup that will attempt to map the result of the lookup to a slice.
func (l lens) L(data any) ([]any, error) {
	answer, err := l.Lookup(data)

	if err != nil {
		return nil, err
	}

	listAnswer, ok := answer.([]any)

	if !ok {
		return nil, NewError(nil, "expected a []any type")
	}

	return listAnswer, nil
}

// LFunc is an alias for Lookup that will attempt to map the result of the lookup to a slice and wrap
// the results in a callback.
func (l lens) LFunc(data any) func() ([]any, error) {
	return func() ([]any, error) {
		return l.L(data)
	}
}

// M is an alias for Lookup that will attempt to map the result of the lookup to a map[string]any.
func (l lens) M(data any) (map[string]any, error) {
	answer, err := l.Lookup(data)

	if err != nil {
		return nil, err
	}

	mapAnswer, ok := answer.(map[string]any)

	if !ok {
		return nil, NewError(nil, "expected a map[string]any type but got a %T instead", answer)
	}

	return mapAnswer, nil
}

// M is an alias for Lookup that will attempt to map the result of the lookup to a map[string]any and wrap
// the results in a callback.
func (l lens) MFunc(data any) func() (map[string]any, error) {
	return func() (map[string]any, error) {
		return l.M(data)
	}
}
