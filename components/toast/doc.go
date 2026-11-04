// Package toast shows a short notice in the corner of the page in answer to
// an htmx request. A handler calls Send after its own response to add one, or
// Respond when the toast is the whole answer to a failed action; for a
// message after a redirect use flash.
package toast
