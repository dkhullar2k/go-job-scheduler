package cronexpr

import (
	"testing"
	"time"
)

func TestParseAndMatchSixFieldEveryTenSeconds(t *testing.T) {
	expression, err := Parse("*/10 * * * * *")
	if err != nil {
		t.Fatal(err)
	}

	for _, second := range []int{0, 10, 20, 30, 40, 50} {
		if !expression.Matches(time.Date(2026, time.September, 30, 12, 34, second, 0, time.FixedZone("test", 5*60*60))) {
			t.Errorf("expected match at second %d", second)
		}
	}
	if expression.Matches(time.Date(2026, time.September, 30, 12, 34, 11, 0, time.UTC)) {
		t.Error("unexpected match at second 11")
	}
}

func TestFiveFieldExpressionRunsAtSecondZero(t *testing.T) {
	expression, err := Parse("34 12 * * *")
	if err != nil {
		t.Fatal(err)
	}
	if !expression.Matches(time.Date(2026, time.September, 30, 12, 34, 0, 0, time.UTC)) {
		t.Error("expected match at second zero")
	}
	if expression.Matches(time.Date(2026, time.September, 30, 12, 34, 1, 0, time.UTC)) {
		t.Error("unexpected match after second zero")
	}
}

func TestParseRejectsOutOfRangeField(t *testing.T) {
	if _, err := Parse("61 * * * *"); err == nil {
		t.Fatal("expected out-of-range minute to fail")
	}
}
