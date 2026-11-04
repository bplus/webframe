# Tutorial

Each step is a complete program in [`examples/`](../../examples) and a page
here explaining only what changed since the step before. Every example is
compiled, tested and linted in CI, and a test checks that each page shows
exactly the code in its example.

1. [Hello, page](01-hello.md): one page in the webframe shell
2. [A list](02-list.md): a sortable, paged table from a Go slice
3. [A form](03-form.md): one text field that submits, saves and says "Saved"
4. [Bad input](04-validation.md): the same form refuses an empty or too short title and shows why
5. [A detail page](05-detail.md): one book on its own page, linked from the list
6. [Edit and delete](06-edit-delete.md): the same form edits a book; delete asks first
7. [The templ version](07-templ.md): step 4 written with `.templ` files instead of plain Go
