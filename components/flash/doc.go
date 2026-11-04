// Package flash carries a one-shot message ("Saved", "Could not delete")
// across a redirect in a short-lived cookie. A handler calls Success, Error,
// Warning or Info before redirecting, and the shell shows the message once on
// the next page when the app sets App.Flash to Take.
package flash
