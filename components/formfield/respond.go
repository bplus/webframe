package formfield

import "net/http"

// RespondFeedback writes the Feedback slot for f with errMsg: the whole
// body of a single-field validation handler (the hx-post target that
// Field.Validate wires). An empty errMsg renders the empty slot, so a
// valid field shows nothing and reserves no space; a message renders
// with role="alert" so it is announced when swapped in. The response is
// text/html with status 200 either way — htmx swaps the Feedback
// wrapper in place (hx-swap="outerHTML"), and a non-2xx status would
// make it skip the swap and leave a stale message.
//
// Typical handler:
//
//	func validateEmail(w http.ResponseWriter, r *http.Request) {
//		msg := ""
//		if taken(r.FormValue("email")) {
//			msg = "That address is already registered"
//		}
//		_ = formfield.RespondFeedback(w, r, emailField, msg)
//	}
func RespondFeedback(w http.ResponseWriter, r *http.Request, f Field, errMsg string) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return Feedback(f.FeedbackID(), errMsg, "").Render(r.Context(), w)
}
