package action

import (
	"errors"
	"fmt"
)

// ErrInvalid is wrapped by every error Validate returns, so a caller can
// tell a programming mistake in a declaration from a render failure with
// errors.Is.
var ErrInvalid = errors.New("invalid action")

// Validate reports an Action whose fields contradict each other. Both
// shapes render without complaint, so Validate is how they are caught:
//
//   - Submit with an Hx* request field (HxGet, HxPost, HxDelete): the
//     button would both submit its form and send its own request. Put the
//     request on the form instead.
//   - Form on a plain link (Href with no Submit and no Hx* field): a link
//     has no form attribute, so Form would be dropped.
//
// The page types (listpage, detailpage, formpage, settingspage,
// dashboardpage, authpage) call it from Respond and return the error
// before writing anything.
func Validate(a Action) error {
	if a.Submit {
		for _, f := range []struct{ name, value string }{
			{"HxGet", a.HxGet}, {"HxPost", a.HxPost}, {"HxDelete", a.HxDelete},
		} {
			if f.value != "" {
				return fmt.Errorf("%w %q: Submit with %s would both submit the form and send its own request; put the request on the form instead", ErrInvalid, a.Label, f.name)
			}
		}
	}
	if a.Form != "" && a.IsLink() {
		return fmt.Errorf("%w %q: Form %q on a plain link is dropped (a link has no form attribute); use Submit or an Hx* request to make it a button", ErrInvalid, a.Label, a.Form)
	}
	return nil
}

// ValidateAll runs Validate on every action in every group and joins the
// errors, or returns nil when all are valid.
func ValidateAll(groups ...[]Action) error {
	var errs []error
	for _, g := range groups {
		for _, a := range g {
			if err := Validate(a); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}
