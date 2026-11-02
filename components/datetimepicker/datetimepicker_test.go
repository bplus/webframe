package datetimepicker

import (
	"bytes"
	"context"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/a-h/templ"
)

func render(t *testing.T, c templ.Component) string {
	t.Helper()
	var buf bytes.Buffer
	if err := c.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

func mustContain(t *testing.T, out string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func mustNotContain(t *testing.T, out string, nots ...string) {
	t.Helper()
	for _, not := range nots {
		if strings.Contains(out, not) {
			t.Errorf("output has unexpected %q:\n%s", not, out)
		}
	}
}

// boolAttr matches a bare boolean attribute.
func boolAttr(name string) *regexp.Regexp {
	return regexp.MustCompile(`\s` + name + `[\s/>]`)
}

var sample = time.Date(2026, 10, 31, 9, 5, 7, 0, time.UTC)

func TestDateTimePicker_TypePerKind(t *testing.T) {
	for _, tc := range []struct {
		kind Kind
		typ  string
	}{
		{"", "date"},
		{KindDate, "date"},
		{KindDateTime, "datetime-local"},
		{KindTime, "time"},
		{Kind("bogus"), "date"},
	} {
		out := render(t, DateTimePicker(Props{Name: "when", Kind: tc.kind}))
		mustContain(t, out, `type="`+tc.typ+`"`, `name="when"`, `id="datetimepicker-when"`, "w-full", "scheme-light", "[[data-theme=dark]_&amp;]:scheme-dark")
	}
}

func TestDateTimePicker_ValueFormatting(t *testing.T) {
	for _, tc := range []struct {
		kind  Kind
		step  int
		value string
	}{
		{KindDate, 0, "2026-10-31"},
		{KindDateTime, 0, "2026-10-31T09:05"},
		{KindTime, 0, "09:05"},
		{KindTime, 1, "09:05:07"},
		{KindDateTime, 15, "2026-10-31T09:05:07"},
		{KindDateTime, 300, "2026-10-31T09:05"},
	} {
		out := render(t, DateTimePicker(Props{Name: "when", Kind: tc.kind, Value: sample, Step: tc.step}))
		mustContain(t, out, `value="`+tc.value+`"`)
	}
}

func TestDateTimePicker_ZeroValueIsEmpty(t *testing.T) {
	out := render(t, DateTimePicker(Props{Name: "when"}))
	mustNotContain(t, out, "value=")
}

func TestDateTimePicker_ValueStringOverrides(t *testing.T) {
	out := render(t, DateTimePicker(Props{Name: "when", Kind: KindDate, Value: sample, ValueString: "not-a-date"}))
	mustContain(t, out, `value="not-a-date"`)
	mustNotContain(t, out, "2026-10-31")
}

func TestDateTimePicker_MinMaxStepRequiredDisabled(t *testing.T) {
	out := render(t, DateTimePicker(Props{
		ID: "d", Name: "when", Kind: KindDate,
		Min: "2026-01-01", Max: "2026-12-31", Step: 7, Required: true, Disabled: true,
	}))
	mustContain(t, out, `id="d"`, `min="2026-01-01"`, `max="2026-12-31"`, `step="7"`)
	for _, attr := range []string{"required", "disabled"} {
		if !boolAttr(attr).MatchString(out) {
			t.Errorf("missing %s attribute:\n%s", attr, out)
		}
	}

	plain := render(t, DateTimePicker(Props{Name: "when"}))
	mustNotContain(t, plain, "min=", "max=", "step=")
	for _, attr := range []string{"required", "disabled"} {
		if boolAttr(attr).MatchString(plain) {
			t.Errorf("unexpected %s attribute:\n%s", attr, plain)
		}
	}
}

func TestDateTimePicker_AttributesWin(t *testing.T) {
	out := render(t, DateTimePicker(Props{Name: "when", Min: "2026-01-01", Attributes: templ.Attributes{"min": "2025-01-01", "aria-describedby": "help"}}))
	mustContain(t, out, `min="2025-01-01"`, `aria-describedby="help"`)
	mustNotContain(t, out, "2026-01-01")
}

func TestControlID(t *testing.T) {
	if got := (Props{ID: "x", Name: "n"}).ControlID(); got != "x" {
		t.Errorf("ID wins: got %q", got)
	}
	if got := (Props{Name: "n"}).ControlID(); got != "datetimepicker-n" {
		t.Errorf("derived from Name: got %q", got)
	}
	if got := (Props{}).ControlID(); got != "" {
		t.Errorf("neither: got %q", got)
	}
}

func TestFormat(t *testing.T) {
	for _, tc := range []struct {
		kind Kind
		want string
	}{
		{KindDate, "2026-10-31"},
		{KindDateTime, "2026-10-31T09:05"},
		{KindTime, "09:05"},
		{"", "2026-10-31"},
	} {
		if got := Format(tc.kind, sample); got != tc.want {
			t.Errorf("Format(%q) = %q, want %q", tc.kind, got, tc.want)
		}
	}
	if got := Format(KindDate, time.Time{}); got != "" {
		t.Errorf("zero time formats as %q, want empty", got)
	}
}

func TestParse_RoundTrip(t *testing.T) {
	loc := time.FixedZone("CST", -6*3600)
	for _, kind := range []Kind{KindDate, KindDateTime, KindTime} {
		in := sample.In(loc)
		got, err := Parse(kind, Format(kind, in), loc)
		if err != nil {
			t.Fatalf("Parse(%q): %v", kind, err)
		}
		if got.Location().String() != loc.String() {
			t.Errorf("Parse(%q) location = %v, want %v", kind, got.Location(), loc)
		}
		switch kind {
		case KindDate:
			if got.Year() != 2026 || got.Month() != 10 || got.Day() != 31 || got.Hour() != 0 {
				t.Errorf("date round trip: %v", got)
			}
		case KindDateTime:
			if !got.Equal(in.Truncate(time.Minute)) {
				t.Errorf("datetime round trip: %v want %v", got, in.Truncate(time.Minute))
			}
		case KindTime:
			if got.Hour() != in.Hour() || got.Minute() != in.Minute() || got.Second() != 0 {
				t.Errorf("time round trip: %v", got)
			}
		}
	}
}

func TestParse_AcceptsSecondsAndFractions(t *testing.T) {
	for _, tc := range []struct {
		kind Kind
		in   string
		sec  int
		nsec int
	}{
		{KindDateTime, "2026-10-31T09:05:07", 7, 0},
		{KindDateTime, "2026-10-31T09:05:07.5", 7, 500_000_000},
		{KindTime, "09:05:07", 7, 0},
		{KindTime, "09:05:07.25", 7, 250_000_000},
		{KindTime, " 09:05 ", 0, 0},
	} {
		got, err := Parse(tc.kind, tc.in, nil)
		if err != nil {
			t.Fatalf("Parse(%q, %q): %v", tc.kind, tc.in, err)
		}
		if got.Second() != tc.sec || got.Nanosecond() != tc.nsec {
			t.Errorf("Parse(%q, %q) = %v", tc.kind, tc.in, got)
		}
		if got.Location() != time.UTC {
			t.Errorf("nil loc should be UTC, got %v", got.Location())
		}
	}
}

func TestParse_EmptyAndInvalid(t *testing.T) {
	got, err := Parse(KindDate, "", nil)
	if err != nil || !got.IsZero() {
		t.Errorf("empty: got %v, %v", got, err)
	}
	for _, tc := range []struct {
		kind Kind
		in   string
	}{
		{KindDate, "31/10/2026"},
		{KindDate, "2026-10-31T09:05"},
		{KindDateTime, "2026-10-31"},
		{KindTime, "9am"},
	} {
		if _, err := Parse(tc.kind, tc.in, nil); err == nil {
			t.Errorf("Parse(%q, %q) should fail", tc.kind, tc.in)
		} else if !strings.Contains(err.Error(), tc.in) || !strings.Contains(err.Error(), string(tc.kind)) {
			t.Errorf("error should name the value and kind: %v", err)
		}
	}
}
