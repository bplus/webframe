// Package datetimepicker renders the browser's native date, date-and-time
// and time inputs through the input component, with a time.Time formatted
// the way each control expects and a Parse for the handler side. There is no
// custom calendar or clock: the platform picker is the picker, which is what
// keeps it usable on a phone.
//
// Form contract: the control submits Name=<value> in its own wire format,
// "2006-01-02" for KindDate, "2006-01-02T15:04" for KindDateTime and "15:04"
// for KindTime (with ":05" seconds appended when Step is not a whole number
// of minutes). The handler calls Parse(kind, r.FormValue(name), loc) with the
// user's location; an empty value parses to the zero time without error.
//
// Phone width (390px): the input is full width (the input component is
// w-full) and a tap opens the platform's native picker, the wheel sheet on
// iOS and the material dialog on Android, so nothing custom has to fit the
// screen. The font size stays at 16px below md so iOS Safari does not zoom
// the page when the control is focused. The picker icon follows the
// control's color-scheme, which the component sets from the theme attribute
// (light by default, dark under [data-theme="dark"]).
package datetimepicker

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/components/input"
)

// Kind selects which native control renders and which wire format the value
// uses. The zero value renders as KindDate.
type Kind string

const (
	// KindDate is <input type="date">; wire format "2006-01-02".
	KindDate Kind = "date"
	// KindDateTime is <input type="datetime-local">; wire format
	// "2006-01-02T15:04" (no zone: the value is wall-clock time in the
	// user's location, which Parse takes as an argument).
	KindDateTime Kind = "datetime-local"
	// KindTime is <input type="time">; wire format "15:04".
	KindTime Kind = "time"
)

// Props configures DateTimePicker. Name is the submitted field; everything
// else is optional. Class merges after the input's built-in classes;
// Attributes spread onto the <input> (after min/max/step/required, so a
// caller's attribute wins on conflict).
type Props struct {
	// ID is the input's id. When empty, ControlID derives one from Name;
	// with neither, the input component generates a random id.
	ID         string
	Class      string
	Attributes templ.Attributes

	Name string
	Kind Kind
	// Value is formatted per Kind with Format (seconds included when Step
	// asks for them). The zero time renders an empty control.
	Value time.Time
	// ValueString, when non-empty, is written to the control verbatim
	// instead of Value: use it to echo what the user submitted on a failed
	// POST without parsing it first.
	ValueString string
	// Min and Max are in the control's wire format (see Kind) and become
	// the min/max attributes, which the picker and browser validation
	// enforce.
	Min, Max string
	// Step is the step attribute: seconds for KindDateTime and KindTime (a
	// value that is not a multiple of 60 makes the control show seconds and
	// Format write them), days for KindDate. 0 omits the attribute (the
	// browser default is one minute, or one day).
	Step     int
	Required bool
	Disabled bool
}

// kind returns the effective Kind: KindDate for the zero value and for any
// unknown value.
func (p Props) kind() Kind {
	switch p.Kind {
	case KindDateTime, KindTime:
		return p.Kind
	default:
		return KindDate
	}
}

// ControlID is the id of the rendered <input>: ID when set, otherwise
// "datetimepicker-"+Name when Name is set, otherwise "" (the input component
// then generates a random id, so supply ID or Name whenever a label or a
// test needs a stable one).
func (p Props) ControlID() string {
	if p.ID != "" {
		return p.ID
	}
	if p.Name != "" {
		return "datetimepicker-" + p.Name
	}
	return ""
}

// inputType maps the Kind onto the input component's Type.
func (p Props) inputType() input.Type {
	switch p.kind() {
	case KindDateTime:
		return input.TypeDateTime
	case KindTime:
		return input.TypeTime
	default:
		return input.TypeDate
	}
}

// value is the string written to the control: ValueString when set, else
// Value formatted for the Kind (with seconds when Step asks for them).
func (p Props) value() string {
	if p.ValueString != "" {
		return p.ValueString
	}
	return format(p.kind(), p.Value, p.showSeconds())
}

// showSeconds reports whether the control shows (and therefore submits)
// seconds: a time-bearing Kind with a Step that is not a whole number of
// minutes.
func (p Props) showSeconds() bool {
	return p.kind() != KindDate && p.Step > 0 && p.Step%60 != 0
}

// attributes builds min/max/step/required and merges the caller's
// Attributes last so they win on conflict.
func (p Props) attributes() templ.Attributes {
	attrs := templ.Attributes{}
	if p.Min != "" {
		attrs["min"] = p.Min
	}
	if p.Max != "" {
		attrs["max"] = p.Max
	}
	if p.Step > 0 {
		attrs["step"] = strconv.Itoa(p.Step)
	}
	if p.Required {
		attrs["required"] = true
	}
	for k, v := range p.Attributes {
		attrs[k] = v
	}
	return attrs
}

// layout is the Go time layout for a Kind, with or without seconds. KindDate
// never has seconds.
func layout(kind Kind, seconds bool) string {
	switch kind {
	case KindDateTime:
		if seconds {
			return "2006-01-02T15:04:05"
		}
		return "2006-01-02T15:04"
	case KindTime:
		if seconds {
			return "15:04:05"
		}
		return "15:04"
	default:
		return "2006-01-02"
	}
}

// format writes t for the control; the zero time is "".
func format(kind Kind, t time.Time, seconds bool) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(layout(kind, seconds))
}

// Format writes t in the wire format of kind: "2006-01-02", "2006-01-02T15:04"
// or "15:04" (minute precision; seconds are dropped, matching the controls'
// default one-minute step). The zero time formats as "", which renders an
// empty control. t is formatted in its own location: convert it to the
// user's location first when it is stored in UTC.
func Format(kind Kind, t time.Time) string {
	return format(Kind(kindOrDate(kind)), t, false)
}

// kindOrDate normalizes an unknown Kind to KindDate.
func kindOrDate(kind Kind) Kind {
	return Props{Kind: kind}.kind()
}

// parseLayouts lists the accepted wire formats per Kind, most precise first:
// browsers may submit seconds (and fractional seconds) when Step asks for
// them, and the HTML spec allows them in every time-bearing value.
func parseLayouts(kind Kind) []string {
	switch kind {
	case KindDateTime:
		return []string{"2006-01-02T15:04:05.999999999", "2006-01-02T15:04:05", "2006-01-02T15:04"}
	case KindTime:
		return []string{"15:04:05.999999999", "15:04:05", "15:04"}
	default:
		return []string{"2006-01-02"}
	}
}

// Parse reads a submitted value back into a time.Time in loc (nil means
// time.UTC). It accepts every form the native control can submit for kind,
// including optional seconds and fractional seconds. An empty or blank s
// returns the zero time and no error, so an optional field needs no special
// case; check IsZero. KindTime values land on January 1 of year 0 in loc,
// as time.Parse does. Any other input returns an error naming the kind and
// the value.
func Parse(kind Kind, s string, loc *time.Location) (time.Time, error) {
	kind = kindOrDate(kind)
	if loc == nil {
		loc = time.UTC
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, nil
	}
	var firstErr error
	for _, l := range parseLayouts(kind) {
		t, err := time.ParseInLocation(l, s, loc)
		if err == nil {
			return t, nil
		}
		if firstErr == nil {
			firstErr = err
		}
	}
	return time.Time{}, fmt.Errorf("datetimepicker: %q is not a valid %s value: %w", s, kind, firstErr)
}
