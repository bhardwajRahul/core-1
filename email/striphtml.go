package email

import (
	"bytes"
	"html"
	"strings"
	"text/template"
)

// StripHTML returns a version of a string with no HTML tags.
func StripHTML(s string) string {
	output := ""

	// if we have a full html page we only need the body
	startBody := strings.Index(s, "<body")
	if startBody > -1 {
		endBody := strings.Index(s, "</body>")
		// try to find the end of the <body tag
		for i := startBody; i < endBody; i++ {
			if s[i] == '>' {
				startBody = i
				break
			}
		}

		if startBody < endBody {
			s = s[startBody:endBody]
		}
	}

	// Shortcut strings with no tags in them
	if !strings.ContainsAny(s, "<>") {
		output = s
	} else {
		// Removing line feeds
		s = strings.ReplaceAll(s, "\n", "")

		// Then replace line breaks with newlines, to preserve that formatting
		s = strings.ReplaceAll(s, "</h1>", "\n\n")
		s = strings.ReplaceAll(s, "</h2>", "\n\n")
		s = strings.ReplaceAll(s, "</h3>", "\n\n")
		s = strings.ReplaceAll(s, "</h4>", "\n\n")
		s = strings.ReplaceAll(s, "</h5>", "\n\n")
		s = strings.ReplaceAll(s, "</h6>", "\n\n")
		s = strings.ReplaceAll(s, "</p>", "\n")
		s = strings.ReplaceAll(s, "<br>", "\n")
		s = strings.ReplaceAll(s, "<br/>", "\n")
		s = strings.ReplaceAll(s, "<br />", "\n")

		// Walk through the string removing all tags
		b := bytes.NewBufferString("")
		inTag := false
		for _, r := range s {
			switch r {
			case '<':
				inTag = true
			case '>':
				inTag = false
			default:
				if !inTag {
					b.WriteRune(r)
				}
			}
		}
		output = b.String()
	}

	// Remove a few common harmless entities, to arrive at something more like plain text
	output = strings.ReplaceAll(output, "&#8216;", "'")
	output = strings.ReplaceAll(output, "&#8217;", "'")
	output = strings.ReplaceAll(output, "&#8220;", "\"")
	output = strings.ReplaceAll(output, "&#8221;", "\"")
	output = strings.ReplaceAll(output, "&nbsp;", " ")
	output = strings.ReplaceAll(output, "&quot;", "\"")
	output = strings.ReplaceAll(output, "&apos;", "'")

	// Translate some entities into their plain text equivalent (for example accents, if encoded as entities)
	output = html.UnescapeString(output)

	// In case we have missed any tags above, escape the text - removes <, >, &, ' and ".
	output = template.HTMLEscapeString(output)

	// After processing, remove some harmless entities &, ' and " which are encoded by HTMLEscapeString
	output = strings.ReplaceAll(output, "&#34;", "\"")
	output = strings.ReplaceAll(output, "&#39;", "'")
	output = strings.ReplaceAll(output, "&amp; ", "& ")     // NB space after
	output = strings.ReplaceAll(output, "&amp;amp; ", "& ") // NB space after

	return output
}
