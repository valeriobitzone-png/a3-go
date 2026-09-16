// SPDX-License-Identifier: Apache-2.0
// Part of the A3 universe. See LICENSE.
// Package jcs implements RFC 8785 JSON Canonicalization Scheme.
package jcs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"unicode/utf16"
)

// Canonicalize returns RFC 8785 bytes for a JSON document.
func Canonicalize(input []byte) ([]byte, error) {
	var v any
	if err := json.Unmarshal(input, &v); err != nil {
		return nil, err
	}
	return CanonicalValue(v)
}

// CanonicalValue serializes a decoded JSON value (maps, slices, floats, strings).
func CanonicalValue(v any) ([]byte, error) {
	var buf bytes.Buffer
	if err := write(&buf, v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func write(buf *bytes.Buffer, v any) error {
	switch x := v.(type) {
	case nil:
		buf.WriteString("null")
	case bool:
		if x {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case float64:
		s, err := es6Number(x)
		if err != nil {
			return err
		}
		buf.WriteString(s)
	case json.Number:
		f, err := x.Float64()
		if err != nil {
			return err
		}
		s, err := es6Number(f)
		if err != nil {
			return err
		}
		buf.WriteString(s)
	case int:
		s, err := es6Number(float64(x))
		if err != nil {
			return err
		}
		buf.WriteString(s)
	case int64:
		s, err := es6Number(float64(x))
		if err != nil {
			return err
		}
		buf.WriteString(s)
	case string:
		buf.WriteString(escape(x))
	case []any:
		buf.WriteByte('[')
		for i, el := range x {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := write(buf, el); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return utf16Less(keys[i], keys[j]) })
		buf.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			buf.WriteString(escape(k))
			buf.WriteByte(':')
			if err := write(buf, x[k]); err != nil {
				return err
			}
		}
		buf.WriteByte('}')
	default:
		return fmt.Errorf("jcs: unsupported type %T", v)
	}
	return nil
}

func utf16Less(a, b string) bool {
	ua := utf16.Encode([]rune(a))
	ub := utf16.Encode([]rune(b))
	n := len(ua)
	if len(ub) < n {
		n = len(ub)
	}
	for i := 0; i < n; i++ {
		if ua[i] != ub[i] {
			return ua[i] < ub[i]
		}
	}
	return len(ua) < len(ub)
}

func escape(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 2)
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if r < 0x20 {
				fmt.Fprintf(&b, `\u%04x`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}

// es6Number implements ECMAScript NumberToString as required by RFC 8785 §3.2.2.3.
func es6Number(f float64) (string, error) {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return "", fmt.Errorf("jcs: invalid JSON number")
	}
	if f == 0 {
		return "0", nil
	}
	sign := ""
	if f < 0 {
		sign = "-"
		f = -f
	}
	eform := strconv.FormatFloat(f, 'e', -1, 64)
	i := strings.IndexByte(eform, 'e')
	if i < 0 {
		return "", fmt.Errorf("jcs: missing exponent in %q", eform)
	}
	coeff := eform[:i]
	exp, err := strconv.Atoi(eform[i+1:])
	if err != nil {
		return "", err
	}
	var digits string
	if dot := strings.IndexByte(coeff, '.'); dot >= 0 {
		digits = coeff[:dot] + coeff[dot+1:]
	} else {
		digits = coeff
	}
	k := len(digits)
	n := exp + 1
	var body string
	switch {
	case k <= n && n <= 21:
		body = digits + strings.Repeat("0", n-k)
	case 0 < n && n <= 21:
		body = digits[:n] + "." + digits[n:]
	case -6 < n && n <= 0:
		body = "0." + strings.Repeat("0", -n) + digits
	default:
		expStr := strconv.Itoa(n - 1)
		if n-1 > 0 {
			expStr = "+" + expStr
		}
		if k == 1 {
			body = digits + "e" + expStr
		} else {
			body = digits[:1] + "." + digits[1:] + "e" + expStr
		}
	}
	return sign + body, nil
}
