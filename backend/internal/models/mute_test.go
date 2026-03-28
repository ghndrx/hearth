package models

import (
	"testing"
	"time"
)

func TestTimeRangeAddDuration(t *testing.T) {
	base := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	tr := TimeRange{
		Start: base,
		End:   base,
	}

	t.Run("add minutes", func(t *testing.T) {
		result := tr.AddDuration("minutes", 30)
		expected := base.Add(30 * time.Minute)
		if !result.Equal(expected) {
			t.Errorf("expected %v, got %v", expected, result)
		}
	})

	t.Run("add zero minutes", func(t *testing.T) {
		result := tr.AddDuration("minutes", 0)
		if !result.Equal(base) {
			t.Errorf("expected %v, got %v", base, result)
		}
	})

	t.Run("unsupported unit returns End unchanged", func(t *testing.T) {
		result := tr.AddDuration("hours", 5)
		if !result.Equal(base) {
			t.Errorf("expected End %v for unsupported unit, got %v", base, result)
		}
	})

	t.Run("empty unit returns End", func(t *testing.T) {
		result := tr.AddDuration("", 10)
		if !result.Equal(base) {
			t.Errorf("expected End %v for empty unit, got %v", base, result)
		}
	})
}

func TestTimeRangeNow(t *testing.T) {
	tr := TimeRange{}
	before := time.Now()
	result := tr.Now()
	after := time.Now()

	if result.End.Before(before) || result.End.After(after) {
		t.Errorf("Now().End should be close to current time")
	}
}

func TestTimeRangeIsAfter(t *testing.T) {
	t1 := TimeRange{End: time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)}
	t2 := TimeRange{End: time.Date(2024, 6, 1, 11, 0, 0, 0, time.UTC)}
	t3 := TimeRange{End: time.Date(2024, 6, 1, 13, 0, 0, 0, time.UTC)}

	if !t1.IsAfter(t2) {
		t.Error("t1 (12:00) should be after t2 (11:00)")
	}
	if t1.IsAfter(t3) {
		t.Error("t1 (12:00) should not be after t3 (13:00)")
	}
	if t1.IsAfter(t1) {
		t.Error("t1 should not be after itself")
	}
}
