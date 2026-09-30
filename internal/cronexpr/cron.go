// Package cronexpr parses and matches five- or six-field numeric cron expressions
// used by the scheduler.
package cronexpr

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type field struct {
	values   map[int]struct{}
	wildcard bool
}

type Expression struct {
	second  field
	minute  field
	hour    field
	day     field
	month   field
	weekday field
}

// Parse accepts five fields (minute through weekday) or six fields (second
// through weekday). Five-field expressions run when seconds equal zero.
func Parse(expression string) (Expression, error) {
	parts := strings.Fields(expression)
	if len(parts) != 5 && len(parts) != 6 {
		return Expression{}, fmt.Errorf("cron expression must have five or six fields, got %d", len(parts))
	}
	limits := [][2]int{{0, 59}, {0, 59}, {0, 23}, {1, 31}, {1, 12}, {0, 7}}
	if len(parts) == 5 {
		limits = limits[1:]
	}
	parsed := make([]field, len(parts))
	for i, part := range parts {
		fieldIndex := i
		if len(parts) == 5 {
			fieldIndex++
		}
		f, err := parseField(part, limits[i][0], limits[i][1], fieldIndex == 5)
		if err != nil {
			return Expression{}, fmt.Errorf("cron field %d (%q): %w", i+1, part, err)
		}
		parsed[i] = f
	}
	if len(parsed) == 5 {
		return Expression{second: parseZeroField(), minute: parsed[0], hour: parsed[1], day: parsed[2], month: parsed[3], weekday: parsed[4]}, nil
	}
	return Expression{second: parsed[0], minute: parsed[1], hour: parsed[2], day: parsed[3], month: parsed[4], weekday: parsed[5]}, nil
}

func parseZeroField() field { return field{values: map[int]struct{}{0: {}}} }

// parseField expands numeric wildcards, comma lists, ranges, and step values
// into the set of values that can match a time component.
func parseField(source string, minimum, maximum int, sundaySeven bool) (field, error) {
	result := field{values: make(map[int]struct{}), wildcard: source == "*"}
	for _, item := range strings.Split(source, ",") {
		if item == "" {
			return field{}, fmt.Errorf("empty list item")
		}
		base, stepText, hasStep := strings.Cut(item, "/")
		if hasStep && strings.Contains(stepText, "/") {
			return field{}, fmt.Errorf("more than one step separator")
		}
		step := 1
		if hasStep {
			value, err := strconv.Atoi(stepText)
			if err != nil || value <= 0 {
				return field{}, fmt.Errorf("step must be a positive integer")
			}
			step = value
		}

		start, end := minimum, maximum
		switch {
		case base == "*":
		case strings.Contains(base, "-"):
			rangeParts := strings.Split(base, "-")
			if len(rangeParts) != 2 {
				return field{}, fmt.Errorf("invalid range %q", base)
			}
			var err error
			start, err = strconv.Atoi(rangeParts[0])
			if err != nil {
				return field{}, fmt.Errorf("invalid range start")
			}
			end, err = strconv.Atoi(rangeParts[1])
			if err != nil {
				return field{}, fmt.Errorf("invalid range end")
			}
		case hasStep:
			start, _ = strconv.Atoi(base)
		default:
			value, err := strconv.Atoi(base)
			if err != nil {
				return field{}, fmt.Errorf("expected a number, range, or wildcard")
			}
			start, end = value, value
		}
		if start < minimum || start > maximum || end < minimum || end > maximum || start > end {
			return field{}, fmt.Errorf("value/range must be between %d and %d", minimum, maximum)
		}
		for value := start; value <= end; value += step {
			if sundaySeven && value == 7 {
				result.values[0] = struct{}{}
			} else {
				result.values[value] = struct{}{}
			}
		}
	}
	fieldSize := maximum - minimum + 1
	if sundaySeven {
		fieldSize--
	}
	result.wildcard = len(result.values) == fieldSize
	return result, nil
}

func (e Expression) Matches(t time.Time) bool {
	t = t.UTC()
	if !has(e.second, t.Second()) || !has(e.minute, t.Minute()) || !has(e.hour, t.Hour()) || !has(e.month, int(t.Month())) {
		return false
	}
	dayMatches := has(e.day, t.Day())
	weekdayMatches := has(e.weekday, int(t.Weekday()))
	// Cron treats day-of-month and weekday as an OR when both are restricted.
	switch {
	case e.day.wildcard && e.weekday.wildcard:
		return true
	case e.day.wildcard:
		return weekdayMatches
	case e.weekday.wildcard:
		return dayMatches
	default:
		return dayMatches || weekdayMatches
	}
}

func has(f field, value int) bool {
	_, ok := f.values[value]
	return ok
}
