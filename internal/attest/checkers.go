package attest

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/tidwall/gjson"
)

// Checker is a composable predicate used in assertions to validate actual values
// against expected conditions.
type Checker[T any] interface {
	// Check returns true if actual satisfies this checker's condition.
	Check(actual T) bool
	// Expected returns a human-readable description of what was expected.
	Expected() string
}

// isChecker validates exact value matching.
type isChecker[T comparable] struct {
	value T
}

// Is creates a checker that validates exact equality.
func Is[T comparable](value T) isChecker[T] {
	return isChecker[T]{value: value}
}

func (m isChecker[T]) Check(actual T) bool {
	return actual == m.value
}

func (m isChecker[T]) Expected() string {
	return fmt.Sprintf("%v", m.value)
}

// isNullChecker validates that a value is nil.
type isNullChecker[T any] struct{}

// IsNull creates a checker that checks if a value is nil.
func IsNull[T comparable]() isNullChecker[T] {
	return isNullChecker[T]{}
}

func (m isNullChecker[T]) Check(actual T) bool {
	v := reflect.ValueOf(actual)
	if !v.IsValid() {
		return true
	}

	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

func (m isNullChecker[T]) Expected() string {
	return "null"
}

// containsChecker validates that a string contains a substring.
type containsChecker struct {
	substring string
}

// Contains creates a checker that checks if actual contains the substring.
func Contains(substring string) containsChecker {
	return containsChecker{substring: substring}
}

func (m containsChecker) Check(actual string) bool {
	return strings.Contains(actual, m.substring)
}

func (m containsChecker) Expected() string {
	return fmt.Sprintf("containing %q", m.substring)
}

// matchesChecker validates that a string matches a regex pattern.
type matchesChecker struct {
	pattern *regexp.Regexp
	raw     string
}

// Matches creates a checker that checks if actual matches the regex pattern.
func Matches(pattern string) matchesChecker {
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		panic(fmt.Sprintf("invalid regex pattern %q: %v", pattern, err))
	}
	return matchesChecker{pattern: compiled, raw: pattern}
}

func (m matchesChecker) Check(actual string) bool {
	return m.pattern.MatchString(actual)
}

func (m matchesChecker) Expected() string {
	return fmt.Sprintf("matching pattern %q", m.raw)
}

// oneOfChecker validates value is one of several valid values.
type oneOfChecker[T comparable] struct {
	values []T
}

// OneOf creates a checker that accepts any of the provided values.
func OneOf[T comparable](values ...T) oneOfChecker[T] {
	return oneOfChecker[T]{values: values}
}

func (m oneOfChecker[T]) Check(actual T) bool {
	for _, v := range m.values {
		if actual == v {
			return true
		}
	}

	return false
}

func (m oneOfChecker[T]) Expected() string {
	if len(m.values) == 0 {
		return "one of []"
	}

	if len(m.values) <= 5 {
		return fmt.Sprintf("one of %v", m.values)
	}

	// Truncate for readability if too many options
	return fmt.Sprintf("one of [%v, %v, %v, ... and %d more]", m.values[0], m.values[1], m.values[2], len(m.values)-3)
}

// notChecker negates another checker.
type notChecker[T comparable] struct {
	checker Checker[T]
}

// Not creates a checker that negates another checker.
func Not[T comparable](checker Checker[T]) notChecker[T] {
	return notChecker[T]{checker: checker}
}

func (m notChecker[T]) Check(actual T) bool {
	return !m.checker.Check(actual)
}

func (m notChecker[T]) Expected() string {
	return fmt.Sprintf("not %s", m.checker.Expected())
}

// checkAll returns true if all checkers pass for the given value.
// If onFail is provided, it's called with the first failing checker.
func checkAll[T any](value T, checkers []Checker[T], onFail func(Checker[T], T)) bool {
	for _, checker := range checkers {
		if !checker.Check(value) {
			if onFail != nil {
				onFail(checker, value)
			}

			return false
		}
	}

	return true
}

// JSONFieldChecker pairs a gjson path with a checker for that field.
type JSONFieldChecker struct {
	Path    string
	Checker Checker[string]
}

// checkAllJSON returns true if all JSON field checkers pass for the given JSON.
// If onFail is provided, it's called with the first failing checker.
func checkAllJSON(json string, checkers []JSONFieldChecker, onFail func(JSONFieldChecker, any)) bool {
	for _, m := range checkers {
		result := gjson.Get(json, m.Path)
		if _, ok := m.Checker.(isNullChecker[string]); ok {
			value := result.Value()
			if value != nil {
				if onFail != nil {
					onFail(m, value)
				}

				return false
			}
		} else {
			value := result.String()
			if !m.Checker.Check(value) {
				if onFail != nil {
					onFail(m, value)
				}

				return false
			}
		}
	}

	return true
}
