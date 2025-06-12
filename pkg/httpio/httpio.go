package httpio

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"sync"
	"unsafe"
)

const delimiter = '.'

var bytesPool = &sync.Pool{
	New: func() interface{} {
		buf := make([]byte, 0, 64)
		return &buf
	},
}

func Unmarshal(r *http.Request, dest interface{}) error {
	if r.Header.Get("Content-Type") == "application/json" {
		// TODO: make json decoder configurable
		if err := json.NewDecoder(r.Body).Decode(dest); err != nil {
			return err
		}
	}

	v := reflect.ValueOf(dest)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return fmt.Errorf("destination must be a non-nil pointer")
	}
	v = v.Elem()

	buf := bytesPool.Get().(*[]byte)
	defer func() {
		s := *buf
		s = s[:0]
		*buf = s // Copy the stack header with new capacity to the heap
		bytesPool.Put(buf)
	}()
	return decode(&decodeIn{r: r}, v, *buf)
}

type decodeIn struct {
	r             *http.Request
	queryVals     url.Values
	parsedCookies []*http.Cookie
}

func (in *decodeIn) findCookieVal(name string) (string, bool) {
	for _, cookie := range in.parsedCookies {
		if cookie.Name == name {
			return cookie.Value, true
		}
	}
	return "", false
}

func decode(in *decodeIn, v reflect.Value, fullName []byte) error {
	t := v.Type()

	switch t.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			v.Set(reflect.New(t.Elem()))
		}
		return decode(in, v.Elem(), fullName)
	case reflect.Struct:
		for i := range t.NumField() {
			field := t.Field(i)

			name, tagType, ok := findInTag(field)
			if !ok {
				continue
			}

			fieldKind := field.Type.Kind()
			if fieldKind == reflect.Struct {
				fullName = appendWithDelimiter(fullName, name)
				if err := decode(in, v.Field(i), fullName); err != nil {
					return err
				}
				fullName = popWithDelimiter(fullName, name)
				continue
			}

			fullName = append(fullName, name...)
			value, ok := getValue(in, fullName, tagType)
			fullName = fullName[:len(fullName)-len(name)]
			if !ok {
				continue
			}

			if err := setField(v.Field(i), value); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unsupported type: %v", t.Kind())
	}

	return nil
}

type tagType int

const (
	tagTypeNone tagType = iota
	tagTypeQuery
	tagTypePath
	tagTypeHeader
	tagTypeCookie
)

func findInTag(t reflect.StructField) ([]byte, tagType, bool) {
	// Check for direct tag names: query, path, header, cookie
	if tag, ok := t.Tag.Lookup("query"); ok && tag != "" {
		return stringBytes(tag), tagTypeQuery, true
	}
	if tag, ok := t.Tag.Lookup("path"); ok && tag != "" {
		return stringBytes(tag), tagTypePath, true
	}
	if tag, ok := t.Tag.Lookup("header"); ok && tag != "" {
		return stringBytes(tag), tagTypeHeader, true
	}
	if tag, ok := t.Tag.Lookup("cookie"); ok && tag != "" {
		return stringBytes(tag), tagTypeCookie, true
	}

	return nil, 0, false
}

type pathLookuper func(r *http.Request, name string) (string, bool)

func defaultPathLookuper(r *http.Request, name string) (string, bool) {
	return "", false
}

var currentPathLookuper pathLookuper = defaultPathLookuper

// SetPathLookuper sets the path lookuper function.
// It is not thread-safe and should be called at the beginning of the program.
func SetPathLookuper(lookuper pathLookuper) {
	currentPathLookuper = lookuper
}

func getValue(in *decodeIn, name []byte, tagType tagType) (string, bool) {
	switch tagType {
	case tagTypeQuery:
		if in.queryVals == nil {
			in.queryVals = in.r.URL.Query()
		}
		vals, ok := in.queryVals[bytesString(name)]
		if !ok || len(vals) == 0 {
			return "", false
		}
		return vals[0], true
	case tagTypePath:
		return currentPathLookuper(in.r, bytesString(name))
	case tagTypeHeader:
		return in.r.Header.Get(bytesString(name)), true
	case tagTypeCookie:
		if cookieVal, ok := in.findCookieVal(bytesString(name)); ok {
			return cookieVal, true
		}
		cookie, err := in.r.Cookie(bytesString(name))
		if err != nil {
			return "", false
		}
		in.parsedCookies = append(in.parsedCookies, cookie)
		return cookie.Value, true
	default:
		return "", false
	}
}

func setField(v reflect.Value, value string) error {
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		return setField(v.Elem(), value)
	}

	switch v.Kind() {
	case reflect.String:
		v.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intVal, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		v.SetInt(intVal)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintVal, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return err
		}
		v.SetUint(uintVal)
	case reflect.Float32, reflect.Float64:
		floatVal, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		v.SetFloat(floatVal)
	case reflect.Bool:
		if value == "true" {
			v.SetBool(true)
		} else {
			v.SetBool(false)
		}
	default:
		return fmt.Errorf("unsupported type: %v", v.Kind())
	}

	return nil
}

func appendWithDelimiter(prefix []byte, name []byte) []byte {
	prefix = append(prefix, name...)
	prefix = append(prefix, delimiter)
	return prefix
}

func popWithDelimiter(prefix []byte, name []byte) []byte {
	return prefix[:len(prefix)-len(name)-1] // -1 for the delimiter
}

//nolint:gosec // TODO: cover with tests
func stringBytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(&s))
}

//nolint:gosec // TODO: cover with tests
func bytesString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
