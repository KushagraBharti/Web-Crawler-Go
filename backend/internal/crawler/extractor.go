package crawler

import (
	"bytes"
	"io"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

type ExtractedContent struct {
	Title string
	Text  string
	Links []string
}

func ExtractContent(base *url.URL, body []byte, maxLinks int) ExtractedContent {
	tok := html.NewTokenizer(bytes.NewReader(body))
	var title string
	var textParts []string
	var links []string
	insideTitle := false
	skipDepth := 0

	for {
		switch tok.Next() {
		case html.ErrorToken:
			if tok.Err() == io.EOF {
				return ExtractedContent{
					Title: strings.TrimSpace(title),
					Text:  compactText(strings.Join(textParts, " ")),
					Links: links,
				}
			}
			return ExtractedContent{
				Title: strings.TrimSpace(title),
				Text:  compactText(strings.Join(textParts, " ")),
				Links: links,
			}
		case html.StartTagToken, html.SelfClosingTagToken:
			name, hasAttr := tok.TagName()
			tag := strings.ToLower(string(name))
			if tag == "script" || tag == "style" || tag == "noscript" {
				skipDepth++
			}
			if tag == "title" {
				insideTitle = true
			}
			if tag == "a" && hasAttr && (maxLinks <= 0 || len(links) < maxLinks) {
				for {
					key, val, more := tok.TagAttr()
					if string(key) == "href" {
						if href := resolveLink(base, string(val)); href != "" {
							links = append(links, href)
						}
					}
					if !more {
						break
					}
				}
			}
		case html.EndTagToken:
			name, _ := tok.TagName()
			tag := strings.ToLower(string(name))
			if tag == "title" {
				insideTitle = false
			}
			if (tag == "script" || tag == "style" || tag == "noscript") && skipDepth > 0 {
				skipDepth--
			}
		case html.TextToken:
			if skipDepth > 0 {
				continue
			}
			text := compactText(string(tok.Text()))
			if text == "" {
				continue
			}
			if insideTitle && title == "" {
				title = text
			}
			textParts = append(textParts, text)
		}
	}
}

func resolveLink(base *url.URL, raw string) string {
	link := strings.TrimSpace(raw)
	if link == "" || strings.HasPrefix(link, "#") || strings.HasPrefix(strings.ToLower(link), "javascript:") || strings.HasPrefix(strings.ToLower(link), "mailto:") {
		return ""
	}
	if strings.HasPrefix(link, "//") && base != nil {
		link = base.Scheme + ":" + link
	}
	parsed, err := url.Parse(link)
	if err != nil {
		return ""
	}
	if base != nil {
		parsed = base.ResolveReference(parsed)
	}
	canonical, _, err := Canonicalize(parsed.String())
	if err != nil {
		return ""
	}
	return canonical
}

func compactText(raw string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(raw)), " ")
}

func buildExcerpt(text string) string {
	clean := compactText(text)
	if len(clean) <= 280 {
		return clean
	}
	return strings.TrimSpace(clean[:280]) + "..."
}
