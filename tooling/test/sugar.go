package test

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/ipfs/gateway-conformance/tooling"
	"github.com/ipfs/gateway-conformance/tooling/check"
	"github.com/ipfs/gateway-conformance/tooling/tmpl"
)

type RequestBuilder struct {
	Method_          string            `json:"method,omitempty"`
	Path_            string            `json:"path,omitempty"`
	Proxy_           string            `json:"proxy,omitempty"`
	UseProxyTunnel_  bool              `json:"useProxyTunnel,omitempty"`
	Headers_         map[string]string `json:"headers,omitempty"`
	FollowRedirects_ bool              `json:"followRedirects,omitempty"`
	Query_           url.Values        `json:"query,omitempty"`
	Body_            []byte            `json:"body,omitempty"`
}

func Request() RequestBuilder {
	return RequestBuilder{
		Method_:          "GET",
		Query_:           make(url.Values),
		FollowRedirects_: false,
	}
}

func Requests(rs ...RequestBuilder) []RequestBuilder {
	return rs
}

func (r RequestBuilder) Path(path string, args ...any) RequestBuilder {
	r.Path_ = tmpl.Fmt(path, args...)
	return r
}

func (r RequestBuilder) Query(key, value string, args ...any) RequestBuilder {
	r.Query_.Add(key, tmpl.Fmt(value, args...))
	return r
}

func (r RequestBuilder) RemoveHeader(hdr string) string {
	if r.Headers_ != nil {
		v, ok := r.Headers_[hdr]
		if ok {
			delete(r.Headers_, v)
			return v
		}
	}
	return ""
}

func (r RequestBuilder) Proxy(path string, args ...any) RequestBuilder {
	r.Proxy_ = tmpl.Fmt(path, args...)
	return r
}

func (r RequestBuilder) WithProxyTunnel() RequestBuilder {
	r.UseProxyTunnel_ = true
	return r
}

func (r RequestBuilder) FollowRedirects() RequestBuilder {
	r.FollowRedirects_ = true
	return r
}

func (r RequestBuilder) Method(method string) RequestBuilder {
	r.Method_ = method
	return r
}

func (r RequestBuilder) Header(k, v string, rest ...any) RequestBuilder {
	if r.Headers_ == nil {
		r.Headers_ = make(map[string]string)
	}

	r.Headers_[k] = tmpl.Fmt(v, rest...)
	return r
}

func (r RequestBuilder) Headers(hs ...HeaderBuilder) RequestBuilder {
	if r.Headers_ == nil {
		r.Headers_ = make(map[string]string)
	}

	for _, h := range hs {
		r.Headers_[h.Key_] = h.Value_
	}

	return r
}

func (r RequestBuilder) Clone() RequestBuilder {
	var clonedHeaders map[string]string
	var clonedQuery map[string][]string

	if r.Headers_ != nil {
		clonedHeaders = make(map[string]string)
		for k, v := range r.Headers_ {
			clonedHeaders[k] = v
		}
	}

	if r.Query_ != nil {
		clonedQuery = make(map[string][]string)
		for k, v := range r.Query_ {
			if v == nil {
				clonedQuery[k] = nil
			} else {
				clonedQueryParams := append([]string{}, v...)
				clonedQuery[k] = clonedQueryParams
			}
		}
	}

	return RequestBuilder{
		Method_:          r.Method_,
		Path_:            r.Path_,
		Proxy_:           r.Proxy_,
		UseProxyTunnel_:  r.UseProxyTunnel_,
		Headers_:         clonedHeaders,
		FollowRedirects_: r.FollowRedirects_,
		Query_:           clonedQuery,
		// TODO: replace this call with bytes.Clone when we switch to Go 1.20
		// See https://github.com/golang/go/issues/45038#issuecomment-799795384
		Body_: append([]byte(nil), r.Body_...),
	}
}

type ExpectValidator interface {
	Validate(t *testing.T, res *http.Response, localReport Reporter)
	Clone() ExpectValidator
}

type ExpectBuilder struct {
	StatusCode_     int             `json:"statusCode,omitempty"`
	StatusCodeFrom_ int             `json:"statusCodeFrom,omitempty"`
	StatusCodeTo_   int             `json:"statusCodeTo,omitempty"`
	Headers_        []HeaderBuilder `json:"headers,omitempty"`
	Body_           interface{}     `json:"body,omitempty"`
	Specs_          []string        `json:"specs,omitempty"`
}

var _ ExpectValidator = (*ExpectBuilder)(nil)

func Expect() ExpectBuilder {
	return ExpectBuilder{Body_: nil}
}

func ResponsesAreEqual() ExpectBuilder {
	return ExpectBuilder{Body_: check.IsEqual}
}

func (e ExpectBuilder) Status(statusCode int) ExpectBuilder {
	e.StatusCode_ = statusCode
	return e
}

func (e ExpectBuilder) StatusMatch(pattern string) ExpectBuilder {
	re := regexp.MustCompile(`^(\d+)(x+)$`)
	matches := re.FindStringSubmatch(pattern)
	if len(matches) != 3 {
		panic("invalid status pattern")
	}

	// Extract the leading digits and the number of 'x' characters
	leadingDigits := matches[1]
	numXs := len(matches[2])

	// Compute the lower bound
	from, err := strconv.Atoi(leadingDigits + strings.Repeat("0", numXs))
	if err != nil {
		panic(fmt.Sprintf("invalid status pattern: %v", err))
	}

	// Compute the upper bound
	to, err := strconv.Atoi(leadingDigits + strings.Repeat("9", numXs))
	if err != nil {
		panic(fmt.Sprintf("invalid status pattern: %v", err))
	}

	e.StatusCodeFrom_ = from
	e.StatusCodeTo_ = to
	return e
}

func (e ExpectBuilder) StatusBetween(from, to int) ExpectBuilder {
	e.StatusCodeFrom_ = from
	e.StatusCodeTo_ = to
	return e
}

func (e ExpectBuilder) Header(h HeaderBuilder) ExpectBuilder {
	e.Headers_ = append(e.Headers_, h)
	return e
}

func (e ExpectBuilder) Spec(spec string) ExpectBuilder {
	e.Specs_ = []string{spec}
	return e
}

func (e ExpectBuilder) Specs(specs ...string) ExpectBuilder {
	e.Specs_ = specs
	return e
}

func (e ExpectBuilder) Bytes(body string) ExpectBuilder {
	e.Body_ = []byte(body)
	return e
}

func (e ExpectBuilder) Headers(hs ...HeaderBuilder) ExpectBuilder {
	e.Headers_ = append(e.Headers_, hs...)
	return e
}

func (e ExpectBuilder) Body(body interface{}) ExpectBuilder {
	switch body := body.(type) {
	case string:
		e.Body_ = []byte(body)
	case []byte:
		e.Body_ = body
	case check.CheckWithHint[string]:
		e.Body_ = body
	case check.CheckWithHint[[]byte]:
		e.Body_ = body
	case check.Check[string]:
		e.Body_ = body
	case check.Check[[]byte]:
		e.Body_ = body
	default:
		panic("body must be string, []byte, or a regular check")
	}

	return e
}

func (e ExpectBuilder) BodyWithHint(hint string, body interface{}) ExpectBuilder {
	switch body := body.(type) {
	case string:
		e.Body_ = check.WithHint[string](
			hint,
			check.IsEqual(body),
		)
	case []byte:
		panic("body with hint for bytes is not implemented yet")
	case check.CheckWithHint[string]:
		panic("this check already has a hint")
	case check.Check[string]:
		e.Body_ = check.WithHint(hint, body)
	default:
		panic("body must be string, []byte, or a regular check")
	}

	return e
}

func (e ExpectBuilder) Validate(t *testing.T, res *http.Response, localReport Reporter) {
	t.Helper()
	tooling.LogSpecs(t, e.Specs_...)

	checks := validateResponse(t, e, res)
	for _, c := range checks {
		t.Run(c.testName, func(t *testing.T) {
			tooling.LogSpecs(t, c.specs...)
			if !c.checkOutput.Success {
				localReport(t, c.checkOutput.Reason)
			}
		})
	}
}

// Clone performs a deep clone of the ExpectBuilder
// Note: if there are [check.Check]s used in the inner header or body components those are only shallowly cloned
func (e ExpectBuilder) Clone() ExpectValidator {
	clone := ExpectBuilder{}
	var clonedHeaders []HeaderBuilder
	for _, h := range e.Headers_ {
		clonedHeaders = append(clonedHeaders, h.Clone())
	}
	clone.StatusCode_ = e.StatusCode_
	clone.Headers_ = clonedHeaders

	if e.Body_ == nil {
		return clone
	}

	switch body := e.Body_.(type) {
	case string:
		clone.Body_ = body
	case []byte:
		clone.Body_ = body[:]
	case check.CheckWithHint[string]:
		clone.Body_ = body
	case check.CheckWithHint[[]byte]:
		clone.Body_ = body
	case check.Check[string]:
		clone.Body_ = body
	case check.Check[[]byte]:
		clone.Body_ = body
	default:
		panic("body must be string, []byte, or a regular check")
	}
	return clone
}

type AllOfExpectBuilder struct {
	Expect_ []ExpectValidator `json:"expect,omitempty"`
}

var _ ExpectValidator = (*AllOfExpectBuilder)(nil)

func AllOf(expect ...ExpectValidator) AllOfExpectBuilder {
	return AllOfExpectBuilder{Expect_: expect}
}

func (e AllOfExpectBuilder) Validate(t *testing.T, res *http.Response, localReport Reporter) {
	t.Helper()

	for i, expect := range e.Expect_ {
		t.Run(
			fmt.Sprintf("Check %d", i), func(t *testing.T) {
				expect.Validate(t, res, localReport)
			})
	}
}

// Clone performs a deep clone of the AllOfExpectBuilder
// Note: if there are [check.Check]s used in the header or body components of the nested builders those are only
// shallowly cloned
func (e AllOfExpectBuilder) Clone() ExpectValidator {
	var clonedInnerValidators []ExpectValidator
	for _, eb := range e.Expect_ {
		clonedInnerValidators = append(clonedInnerValidators, eb.Clone())
	}
	clone := AllOfExpectBuilder{Expect_: clonedInnerValidators}
	return clone
}

type AnyOfExpectBuilder struct {
	Expect_ []ExpectBuilder `json:"expect,omitempty"`
}

var _ ExpectValidator = (*AnyOfExpectBuilder)(nil)

func AnyOf(expect ...ExpectBuilder) AnyOfExpectBuilder {
	return AnyOfExpectBuilder{Expect_: expect}
}

func (e AnyOfExpectBuilder) Validate(t *testing.T, res *http.Response, localReport Reporter) {
	t.Helper()

	if len(e.Expect_) == 0 {
		return
	}

	hadASuccessfulResponse := false
	for i, expect := range e.Expect_ {
		checks := validateResponse(t, expect, res)
		responseSucceeded := true
		for _, c := range checks {
			if !c.checkOutput.Success {
				responseSucceeded = false
				break
			}
		}
		if responseSucceeded {
			hadASuccessfulResponse = true
		}

		t.Run(fmt.Sprintf("Check %d", i),
			func(t *testing.T) {
				if !responseSucceeded {
					for _, c := range checks {
						if c.checkOutput.Success {
							t.Logf("Test %s passed", c.testName)
						} else {
							t.Logf("Test %s failed with: %s", c.testName, c.checkOutput.Reason)
						}
					}
				}
			})
	}

	if !hadASuccessfulResponse {
		localReport(t, "none of the response options were valid")
	}
}

// Clone performs a deep clone of the AnyOfExpectBuilder
// Note: if there are [check.Check]s used in the header or body components of the nested builders those are only
// shallowly cloned
func (e AnyOfExpectBuilder) Clone() ExpectValidator {
	var clonedInnerBuilders []ExpectBuilder
	for _, eb := range e.Expect_ {
		clonedInnerBuilders = append(clonedInnerBuilders, eb.Clone().(ExpectBuilder))
	}
	clone := AnyOfExpectBuilder{Expect_: clonedInnerBuilders}
	return clone
}

type HeaderBuilder struct {
	Key_   string                `json:"key,omitempty"`
	Value_ string                `json:"value,omitempty"`
	Check_ check.Check[[]string] `json:"check,omitempty"`
	Hint_  string                `json:"hint,omitempty"`
	Specs_ []string              `json:"specs,omitempty"`
	Not_   bool                  `json:"not,omitempty"`
}

func Header(key string, rest ...any) HeaderBuilder {
	if len(rest) > 0 {
		// check if rest[0] is a string
		if value, ok := rest[0].(string); ok {
			value := tmpl.Fmt(value, rest[1:]...)
			return HeaderBuilder{Key_: key, Value_: value, Check_: check.IsUniqAnd(check.IsEqual(value))}
		} else {
			panic("rest[0] must be a string")
		}
	}

	return HeaderBuilder{Key_: key}
}

func (h HeaderBuilder) Contains(value string, rest ...any) HeaderBuilder {
	h.Check_ = check.IsUniqAnd(check.Contains(value, rest...))
	return h
}

func (h HeaderBuilder) Matches(value string, rest ...any) HeaderBuilder {
	h.Check_ = check.IsUniqAnd(check.Matches(value, rest...))
	return h
}

func (h HeaderBuilder) Hint(hint string) HeaderBuilder {
	h.Hint_ = hint
	return h
}

func (h HeaderBuilder) Specs(specs ...string) HeaderBuilder {
	h.Specs_ = specs
	return h
}

func (h HeaderBuilder) Spec(spec string) HeaderBuilder {
	h.Specs_ = []string{spec}
	return h
}

func (h HeaderBuilder) Equals(value string, args ...any) HeaderBuilder {
	h.Check_ = check.IsUniqAnd(check.IsEqual(value, args...))
	return h
}

func (h HeaderBuilder) Has(values ...string) HeaderBuilder {
	for _, value := range values {
		h.Check_ = check.IsUniqAnd(check.Contains(value))
	}
	return h
}

func (h HeaderBuilder) IsEmpty() HeaderBuilder {
	h.Check_ = check.CheckIsEmpty{}
	return h
}

func (h HeaderBuilder) Checks(f func(string) bool) HeaderBuilder {
	h.Check_ = check.IsUniqAnd(check.CheckFunc[string]{
		Fn: f,
	})
	return h
}

func (h HeaderBuilder) ChecksAll(f func([]string) bool) HeaderBuilder {
	h.Check_ = check.CheckFunc[[]string]{
		Fn: f,
	}
	return h
}

func (h HeaderBuilder) Not() HeaderBuilder {
	h.Not_ = !h.Not_
	return h
}

func (h HeaderBuilder) Exists() HeaderBuilder {
	return h.Not().IsEmpty()
}

// Clone performs a shallow clone of the HeaderBuilder
// Note: The Check field is an interface and as a result is just copied
func (h HeaderBuilder) Clone() HeaderBuilder {
	clone := HeaderBuilder{
		Key_:   h.Key_,
		Value_: h.Key_,
		Check_: h.Check_,
		Hint_:  h.Hint_,
		Not_:   h.Not_,
	}
	return clone
}

type ExpectsBuilder struct {
	payloadsAreEquals bool
}

func Responses() ExpectsBuilder {
	return ExpectsBuilder{}
}

func (e ExpectsBuilder) HaveTheSamePayload() ExpectsBuilder {
	e.payloadsAreEquals = true
	return e
}
