package formfield

import "net/http"

// RespondFeedback writes the Feedback slot for f with errMsg: the whole
// body of the handler behind Field.Validate. An empty errMsg renders the
// empty slot; a message renders with role="alert". The status is always
// 200, since htmx skips the swap on a non-2xx status and would leave a
// stale message.
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
