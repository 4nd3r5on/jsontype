package jsontype

import (
	"encoding/json"
	"fmt"
	"strings"
)

func detectNumberType(f float64) DetectedType {
	if f == float64(int32(f)) && f >= -2147483648 && f <= 2147483647 {
		return TypeInt32
	}
	if f == float64(int64(f)) {
		return TypeInt64
	}
	return TypeFloat64
}

func detectNumberTypeFromString(s string) DetectedType {
	// Try to parse as int first
	var i int64
	if _, err := fmt.Sscanf(s, "%d", &i); err == nil {
		if i >= -2147483648 && i <= 2147483647 {
			return TypeInt32
		}
		return TypeInt64
	}
	return TypeFloat64
}

func isIntegerKey(key string) bool {
	for _, c := range key {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(key) > 0
}

func PathToString(path []string) string {
	if len(path) == 0 {
		return "$"
	}

	var b strings.Builder
	b.WriteByte('$')

	for _, p := range path {
		switch {
		case p == "":
			// array wildcard
			b.WriteString("[]")
		case isNumeric(p):
			// array index
			b.WriteByte('[')
			b.WriteString(p)
			b.WriteByte(']')
		default:
			// object key
			b.WriteByte('.')
			b.WriteString(p)
		}
	}

	return b.String()
}

func StringToPath(s string) []string {
	if s == "" || s == "$" {
		return []string{}
	}

	s = strings.TrimPrefix(s, "$")

	var out []string
	var buf strings.Builder

	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '.':
			if buf.Len() > 0 {
				out = append(out, buf.String())
				buf.Reset()
			}
		case '[':
			if buf.Len() > 0 {
				out = append(out, buf.String())
				buf.Reset()
			}

			j := i + 1
			for j < len(s) && s[j] != ']' {
				j++
			}

			// [] → wildcard
			if j == i+1 {
				out = append(out, "")
			} else {
				out = append(out, s[i+1:j])
			}

			i = j // skip until ']'
		default:
			buf.WriteByte(s[i])
		}
	}

	if buf.Len() > 0 {
		out = append(out, buf.String())
	}

	return out
}

func InferMergedContainerType(children []*FieldInfo, key string) DetectedType {
	// If all children containing 'key' have the same container type → use it.
	var t DetectedType
	for _, c := range children {
		sub := c.ChildrenMap[key]
		if sub == nil {
			continue
		}
		if t == "" {
			t = sub.Type
		} else if t != sub.Type {
			return TypeObj // fallback: treat as generic obj
		}
	}
	if t == "" {
		return TypeObj
	}
	return t
}

func IsDelim(token json.Token, expectDelim json.Delim) bool {
	delim, isDelim := token.(json.Delim)
	if isDelim && expectDelim == delim {
		return true
	}
	return false
}

func pathMatches(current, target []string) bool {
	minLen := min(len(current), len(target))

	for i := range minLen {
		currentPart := current[i]
		targetPart := target[i]
		if currentPart != targetPart {
			return false
		}
	}
	return true
}

func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
