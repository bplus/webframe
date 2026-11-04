package action

import (
	"errors"
	"strings"
	"testing"
)

// TestValidate_SubmitWithHx: Submit plus any Hx* request field is an
// error naming the action and the field.
func TestValidate_SubmitWithHx(t *testing.T) {
	for field, a := range map[string]Action{
		"HxGet":    {Label: "Save", Submit: true, HxGet: "/x"},
		"HxPost":   {Label: "Save", Submit: true, HxPost: "/x"},
		"HxDelete": {Label: "Save", Submit: true, HxDelete: "/x"},
	} {
		err := Validate(a)
		if err == nil || !errors.Is(err, ErrInvalid) {
			t.Fatalf("%s: err = %v, want ErrInvalid", field, err)
		}
		if !strings.Contains(err.Error(), `"Save"`) || !strings.Contains(err.Error(), field) {
			t.Errorf("%s: message %q should name the action and the field", field, err)
		}
	}
}

// TestValidate_FormOnLink: Form on a plain link is an error; on a button
// (Submit or an Hx* request) it is fine.
func TestValidate_FormOnLink(t *testing.T) {
	err := Validate(Action{Label: "Open", Href: "/x", Form: "bulk"})
	if err == nil || !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), `"Open"`) || !strings.Contains(err.Error(), "Form") {
		t.Fatalf("Form on a link: err = %v", err)
	}
	for _, ok := range []Action{
		{Label: "Delete", HxPost: "/bulk", Form: "bulk"},
		{Label: "Save", Submit: true, Form: "settings"},
		{Label: "Plain", Href: "/x"},
		{Label: "Save", Submit: true},
		{},
	} {
		if err := Validate(ok); err != nil {
			t.Errorf("Validate(%+v) = %v, want nil", ok, err)
		}
	}
}

// TestValidateAll joins every problem and is nil for valid actions.
func TestValidateAll(t *testing.T) {
	if err := ValidateAll([]Action{{Label: "A", Href: "/a"}}, nil); err != nil {
		t.Errorf("valid: %v", err)
	}
	err := ValidateAll(
		[]Action{{Label: "A", Href: "/a", Form: "f"}},
		[]Action{{Label: "B", Submit: true, HxPost: "/b"}},
	)
	if err == nil || !strings.Contains(err.Error(), `"A"`) || !strings.Contains(err.Error(), `"B"`) {
		t.Errorf("joined: %v", err)
	}
}
