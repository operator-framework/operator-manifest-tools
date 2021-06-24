package pullspec

import (
	"strconv"
	"strings"
)


type lensBuilder struct {
	funcs []func(interface{}) (interface{}, error)
	path  []string
}

type lens struct {
	funcs []func(interface{}) (interface{}, error)
}

func NewLens() *lensBuilder {
	return &lensBuilder{
		funcs: []func(interface{}) (interface{}, error){},
		path:  []string{},
	}
}

func (d *lensBuilder) L(i int) *lensBuilder {
	d.path = append(d.path, strconv.Itoa(i))
	d.funcs = append(d.funcs, func(data interface{}) (interface{}, error) {
		localI := i
		slice, ok := data.([]interface{})

		if !ok {
			return nil, NewError(ErrNotFound, "expected a []interface{} type on step %s path %s", strconv.Itoa(localI), strings.Join(d.path, ","))
		}

		if i < 0 || i >= len(slice) {
			return nil, NewError(ErrNotFound, "not found on step %s path %s", strconv.Itoa(localI), strings.Join(d.path, ","))
		}

		return slice[localI], nil
	})

	return d
}

func (d *lensBuilder) M(key string) *lensBuilder {
	d.path = append(d.path, key)
	d.funcs = append(d.funcs, func(data interface{}) (interface{}, error) {
		localKey := key
		mmap, ok := data.(map[string]interface{})

		if !ok {
			return nil, NewError(ErrNotFound, "expected a map[string]interface{} type on step %s path %s", localKey, strings.Join(d.path, ","))
		}

		v, ok := mmap[key]
		if !ok {
			return nil, NewError(ErrNotFound, "not found on step %s path %s", localKey, strings.Join(d.path, ","))
		}

		return v, nil
	})
	return d
}

func (d *lensBuilder) Apply(l lens) *lensBuilder {
	d.path = append(d.path, "*")
	d.funcs = append(d.funcs, func(data interface{}) (interface{}, error) {
		localLens := l
		slice, ok := data.([]interface{})

		if !ok {
			return nil, NewError(ErrNotFound, "expected a []interface{} type on step * path %s", strings.Join(d.path, ","))
		}

		results := make([]interface{}, 0, len(slice))

		for i := range slice {
			data := slice[i]

			result, err := localLens.Lookup(data)

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

func (d *lensBuilder) Build() lens {
	funcs := make([]func(interface{}) (interface{}, error), 0, len(d.funcs))

	for i := range d.funcs {
		localFunc := d.funcs[i]
		funcs = append(funcs, localFunc)
	}

	return lens{
		funcs: funcs,
	}
}

func (l lens) Lookup(data interface{}) (result interface{}, err error) {
	result = data
	for _, fun := range l.funcs {
		result, err = fun(result)

		if err != nil {
			return
		}
	}

	return
}

func (l lens) L(data interface{}) ([]interface{}, error) {
	answer, err := l.Lookup(data)

	if err != nil {
		return nil, err
	}

	listAnswer, ok := answer.([]interface{})

	if !ok {
		return nil, NewError(nil, "expected a []interface{} type")
	}

	return listAnswer, nil
}

func (l lens) LFunc(data interface{}) func() ([]interface{}, error) {
	return func() ([]interface{}, error) {
		return l.L(data)
	}
}

func (l lens) M(data interface{}) (map[string]interface{}, error) {
	answer, err := l.Lookup(data)

	if err != nil {
		return nil, err
	}

	mapAnswer, ok := answer.(map[string]interface{})

	if !ok {
		return nil, NewError(nil, "expected a []interface{} type")
	}

	return mapAnswer, nil
}

func (l lens) MFunc(data interface{}) func() (map[string]interface{}, error) {
	return func() (map[string]interface{}, error) {
		return l.M(data)
	}
}
