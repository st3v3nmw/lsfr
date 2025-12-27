package attest

import (
	"fmt"
	"regexp"
	"strings"
)

// Matcher interface for validating values.
type Matcher[T comparable] interface {
	Matches(actual T) bool
	Expected() string // for error messages
}

// isMatcher validates exact value matching.
type isMatcher[T comparable] struct {
	value T
}

// Is creates a matcher that validates exact equality.
func Is[T comparable](value T) isMatcher[T] {
	return isMatcher[T]{value: value}
}

func (m isMatcher[T]) Matches(actual T) bool {
	return actual == m.value
}

func (m isMatcher[T]) Expected() string {
	return fmt.Sprintf("%v", m.value)
}

// containsMatcher validates that a string contains a substring.
type containsMatcher struct {
	substring string
}

// Contains creates a matcher that checks if actual contains the substring.
func Contains(substring string) containsMatcher {
	return containsMatcher{substring: substring}
}

func (m containsMatcher) Matches(actual string) bool {
	return strings.Contains(actual, m.substring)
}

func (m containsMatcher) Expected() string {
	return fmt.Sprintf("containing %q", m.substring)
}

// matchesMatcher validates that a string matches a regex pattern.
type matchesMatcher struct {
	pattern *regexp.Regexp
	raw     string
}

// Matches creates a matcher that checks if actual matches the regex pattern.
func Matches(pattern string) matchesMatcher {
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		panic(fmt.Sprintf("invalid regex pattern %q: %v", pattern, err))
	}
	return matchesMatcher{pattern: compiled, raw: pattern}
}

func (m matchesMatcher) Matches(actual string) bool {
	return m.pattern.MatchString(actual)
}

func (m matchesMatcher) Expected() string {
	return fmt.Sprintf("matching pattern %q", m.raw)
}

// oneOfMatcher validates value is one of several valid values.
type oneOfMatcher[T comparable] struct {
	values []T
}

// OneOf creates a matcher that accepts any of the provided values.
func OneOf[T comparable](values ...T) oneOfMatcher[T] {
	return oneOfMatcher[T]{values: values}
}

func (m oneOfMatcher[T]) Matches(actual T) bool {
	for _, v := range m.values {
		if actual == v {
			return true
		}
	}

	return false
}

func (m oneOfMatcher[T]) Expected() string {
	if len(m.values) == 0 {
		return "one of []"
	}

	if len(m.values) <= 5 {
		return fmt.Sprintf("one of %v", m.values)
	}

	// Truncate for readability if too many options
	return fmt.Sprintf("one of [%v, %v, %v, ... and %d more]", m.values[0], m.values[1], m.values[2], len(m.values)-3)
}

// notMatcher negates another matcher.
type notMatcher[T comparable] struct {
	matcher Matcher[T]
}

// Not creates a matcher that negates another matcher.
func Not[T comparable](matcher Matcher[T]) notMatcher[T] {
	return notMatcher[T]{matcher: matcher}
}

func (m notMatcher[T]) Matches(actual T) bool {
	return !m.matcher.Matches(actual)
}

func (m notMatcher[T]) Expected() string {
	return fmt.Sprintf("not %s", m.matcher.Expected())
}
