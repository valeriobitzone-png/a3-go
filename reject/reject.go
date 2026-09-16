// SPDX-License-Identifier: Apache-2.0
// Part of the A3 universe. See LICENSE.
// Package reject is the explicit failure type for CORE MUST / MUST NOT.
package reject

import "fmt"

// Error carries a conformance or protocol code. Silence is not a pass.
type Error struct {
	Code   string
	Reason string
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Reason)
}

func New(code, reason string) *Error {
	return &Error{Code: code, Reason: reason}
}

func As(err error) *Error {
	if e, ok := err.(*Error); ok {
		return e
	}
	return nil
}
