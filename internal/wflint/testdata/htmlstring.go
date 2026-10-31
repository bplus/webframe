package x

import "fmt"

// Comments may mention <div class="x"> freely.
func Click(n int) string {
	return fmt.Sprintf(`<button class="btn">%d</button>`, n)
}

/* <table> in a block comment is fine */
var r = '"'
var ok = "<b>bold</b> and <a href=\"/x\">link</a>, x<divider"
var okMention = "<div>"
var json = `{"class":"x"}`
