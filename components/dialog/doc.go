// Package dialog renders a modal dialog with a trigger, header, content,
// footer and close button. Use it for a short task that must be finished or
// dismissed before going on; confirmations go through action.Confirm instead.
//
// ContentProps.Width picks the width tier (512px by default; WidthLg and
// WidthXl match slideover's 640px and 800px). Content is never taller than
// the screen. For a long list, put the list in a Body: the Body scrolls and
// the parts before and after it are pinned (they stay in view).
//
// Needs JavaScript: the trigger does not open the dialog, so its content
// cannot be reached.
package dialog
