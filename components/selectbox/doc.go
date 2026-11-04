// Package selectbox renders a styled select menu, with optional search and
// multiple selection, whose chosen values are submitted through a hidden
// input. In a form, declare a formfield.Field with TypeSelect instead of
// building one by hand.
//
// Needs JavaScript: the hidden input the form submits is filled by the
// script, so without it the field submits an empty value (an edited record
// loses its value, or a required field fails with 422) and the menu does not
// open.
package selectbox
