package pages

import "github.com/bplus/webframe/components/prose"

// proseSpecimen is the typography specimen the showcase renders through
// prose.Prose. It is content the app authored itself, so it is a prose.HTML
// value: the UI lint leaves the markup inside a prose.HTML conversion alone
// (prose is the one sanctioned route for trusted content HTML) and reports
// the same literal anywhere else.
var proseSpecimen = prose.HTML(`
<h2>Typography specimen</h2>
<p>This block is <code>prose.HTML</code>: trusted markup the app authored itself, written verbatim inside the <code>.wf-prose</code> scope. Nothing here carries a class; the headings, the <a href="#prose">links</a>, the <strong>strong</strong> and <em>emphasised</em> runs, the lists and the table below are styled by the scope alone.</p>
<h3>Lists</h3>
<ul>
	<li>An unordered list with a short item,</li>
	<li>a longer item that wraps onto a second line at phone width so the hanging indent can be checked against the bullet,</li>
	<li>and a nested list:
		<ol>
			<li>first,</li>
			<li>second.</li>
		</ol>
	</li>
</ul>
<h3>Table</h3>
<table>
	<thead>
		<tr><th>Item</th><th>Type</th><th>Released</th><th>Rating</th><th>Used in</th></tr>
	</thead>
	<tbody>
		<tr><td>Blade of the Fallen Star</td><td>Weapon</td><td>2026-09-12</td><td>5</td><td>Starfall Pack</td></tr>
		<tr><td>Netherese Lesser Folly</td><td>Weapon</td><td>2026-08-02</td><td>3</td><td>Netherese Relics</td></tr>
		<tr><td>Circlet of Borrowed Thought</td><td>Accessory</td><td>2026-06-28</td><td>2</td><td>—</td></tr>
	</tbody>
</table>
<h3>Quote and code</h3>
<blockquote>
	<p>The item is the unit of work; the pack is the unit of release. Everything else is bookkeeping.</p>
</blockquote>
<p>Inline code such as <code>Items.Create(it)</code> wraps with the sentence; a block keeps its lines:</p>
<pre><code>func (s *ItemStore) Create(it Item) Item {
	it.ID = strconv.Itoa(s.next)
	s.next++
	it.Updated = time.Now()
	s.items = append(s.items, it)
	return it
}</code></pre>
<h4>Image and rule</h4>
<p>An image never exceeds the block's width, and a horizontal rule closes the specimen.</p>
<img src="/static/img/placeholder-wide.svg" alt="A grey 16:9 placeholder" width="640" height="360">
<hr>
<p>A very long unbroken token, https://example.com/items/circlet-of-borrowed-thought/history/versions/0000000000000000000001, wraps instead of widening the page.</p>
`)
