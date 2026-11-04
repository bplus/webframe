// Package autocomplete renders an htmx type-ahead picker: a text box that
// asks a server URL for matches as the user types and stores the chosen
// item's value in a hidden input. Use it in a form when the options are too
// many for a select; the handler answers with Results.
//
// Needs JavaScript: suggestions never load and the hidden value input stays
// empty, so the form submits the typed text with an empty value, and the
// check that blocks a submit with nothing picked does not run.
package autocomplete
