package anypub

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"unicode/utf8"
)

type ImageAsset struct {
	ID     string  `json:"id"`
	Blob   Record  `json:"blob"`
	Alt    string  `json:"alt"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	URL    string  `json:"url"`
}
type MarkdownBlock struct {
	Kind         string `json:"kind"`
	Source       string `json:"source"`
	RawSource    string `json:"rawSource,omitempty"`
	HeadingLevel int    `json:"headingLevel,omitempty"`
	Language     string `json:"language,omitempty"`
	URL          string `json:"url,omitempty"`
	Alt          string `json:"alt,omitempty"`
	ListLevel    int    `json:"listLevel,omitempty"`
	ListStart    int    `json:"listStart,omitempty"`
}
type NativeContentInput struct {
	Host        string          `json:"host"`
	Markdown    string          `json:"markdown"`
	Description string          `json:"description"`
	Assets      []ImageAsset    `json:"assets"`
	Blocks      []MarkdownBlock `json:"blocks"`
}
type NativeContent struct {
	Content     Record `json:"content"`
	TextContent string `json:"textContent"`
}

var unescapeInline = regexp.MustCompile("\\\\([\\\\`*_\\[\\]])")
var linkInline = regexp.MustCompile(`^\[([^\]]+)\]\(([^\s]+)\)`)

func safeURL(s string) bool {
	if !strings.HasPrefix(strings.ToLower(s), "http://") && !strings.HasPrefix(strings.ToLower(s), "https://") {
		return false
	}
	for _, r := range s {
		if r <= 32 {
			return false
		}
	}
	return true
}

// ArticleRichText converts inline Markdown to native facets with UTF-8 byte indices.
func ArticleRichText(source, ns string) Record {
	plain := ""
	facets := []any{}
	var appendText func(string, int)
	appendText = func(s string, depth int) {
		start := 0
		for i := 0; i < len(s); {
			end, inner, kind, uri := 0, "", "", ""
			if s[i] == '`' {
				n := 1
				for i+n < len(s) && s[i+n] == '`' {
					n++
				}
				if j := strings.Index(s[i+n:], strings.Repeat("`", n)); j >= 0 {
					end = i + n + j + n
					inner = s[i+n : i+n+j]
					kind = "code"
				}
			}
			if end == 0 && s[i] == '[' {
				if m := linkInline.FindStringSubmatch(s[i:]); m != nil {
					end = i + len(m[0])
					inner, uri, kind = m[1], m[2], "link"
				}
			}
			if end == 0 {
				for _, d := range []string{"**", "__", "~~", "++", "*", "_"} {
					if !strings.HasPrefix(s[i:], d) {
						continue
					}
					if j := strings.Index(s[i+len(d):], d); j > 0 && !strings.Contains(s[i+len(d):i+len(d)+j], "\n") {
						end = i + len(d) + j + len(d)
						inner = s[i+len(d) : i+len(d)+j]
						kind = "italic"
						if len(d) == 2 {
							kind = "bold"
						}
						if d == "~~" {
							kind = "strikethrough"
						}
						if d == "++" {
							kind = "underline"
						}
						break
					}
				}
			}
			if end == 0 {
				i++
				continue
			}
			plain += unescapeInline.ReplaceAllString(s[start:i], "$1")
			bs := len(plain)
			if kind == "code" || depth >= 8 {
				plain += inner
			} else {
				appendText(inner, depth+1)
			}
			be := len(plain)
			if be > bs && (kind != "link" || safeURL(uri)) {
				f := Record{"$type": ns + "#" + kind}
				if kind == "link" {
					f["uri"] = uri
				}
				facets = append(facets, Record{"index": Record{"byteStart": bs, "byteEnd": be}, "features": []any{f}})
			}
			i = end
			start = end
		}
		plain += unescapeInline.ReplaceAllString(s[start:], "$1")
	}
	appendText(source, 0)
	r := Record{"plaintext": plain}
	if len(facets) > 0 {
		r["facets"] = facets
	}
	return r
}

var headingPrefix = regexp.MustCompile(`^\s{0,3}#{1,6}\s+`)
var quotePrefix = regexp.MustCompile(`^\s*>\s?`)
var listPrefix = regexp.MustCompile(`^\s*(?:[-*+]|\d+\.)\s*`)
var taskPrefix = regexp.MustCompile(`^\[([ xX])\]\s*([\s\S]*)$`)
var assetURL = regexp.MustCompile(`article-asset://[^\s)]+`)
var embedURL = regexp.MustCompile(`@\[embed\]\((https?://[^\s)]+)\)`)

func headingText(s string) string { return headingPrefix.ReplaceAllString(s, "") }
func codeText(s string) string {
	lines := strings.Split(s, "\n")
	if len(lines) > 0 {
		f := strings.TrimSpace(lines[0])
		if strings.HasPrefix(f, "```") || strings.HasPrefix(f, "~~~") {
			lines = lines[1:]
			if len(lines) > 0 && strings.HasPrefix(strings.TrimSpace(lines[len(lines)-1]), f[:3]) {
				lines = lines[:len(lines)-1]
			}
			return strings.Join(lines, "\n")
		}
	}
	return s
}
func blockCodeText(b MarkdownBlock) string {
	if b.RawSource != "" {
		return codeText(b.RawSource)
	}
	return b.Source
}
func blockHeadingText(b MarkdownBlock) string {
	if b.RawSource != "" {
		return headingText(b.RawSource)
	}
	return b.Source
}
func quoteLines(s string) []string {
	a := strings.Split(s, "\n")
	for i := range a {
		a[i] = quotePrefix.ReplaceAllString(a[i], "")
	}
	return a
}
func BuildArticleNativeContent(in NativeContentInput) (NativeContent, error) {
	lookup := map[string]ImageAsset{}
	for _, a := range in.Assets {
		lookup["article-asset://"+a.ID] = a
	}
	resolve := func(u string) (*ImageAsset, error) {
		a, ok := lookup[u]
		if !ok {
			if strings.HasPrefix(u, "article-asset://") {
				return nil, fmt.Errorf("An article image has not been uploaded.")
			}
			if !safeURL(u) {
				return nil, fmt.Errorf("Article media must use an uploaded image or an HTTP(S) URL.")
			}
			return nil, nil
		}
		if math.IsNaN(a.Width) || math.IsNaN(a.Height) || a.Width <= 0 || a.Height <= 0 || a.Width > 9007199254740991 || a.Height > 9007199254740991 || math.Trunc(a.Width) != a.Width || math.Trunc(a.Height) != a.Height || str(a.Blob["$type"]) != "blob" || str(object(a.Blob["ref"])["$link"]) == "" {
			return nil, fmt.Errorf("Invalid article image asset.")
		}
		return &a, nil
	}
	plain := []string{}
	for _, b := range in.Blocks {
		if b.Kind == "image" || b.Kind == "embed" {
			if _, e := resolve(b.URL); e != nil {
				return NativeContent{}, e
			}
		}
		s := b.Source
		switch b.Kind {
		case "image":
			s = b.Alt
		case "embed":
			s = b.URL
		case "thematic-break", "empty":
			s = ""
		case "code":
			s = blockCodeText(b)
		case "heading":
			s = str(ArticleRichText(blockHeadingText(b), "")["plaintext"])
		case "quote":
			s = str(ArticleRichText(strings.Join(quoteLines(s), "\n"), "")["plaintext"])
		default:
			if b.Kind == "ordered-list" || b.Kind == "unordered-list" {
				s = listPrefix.ReplaceAllString(s, "")
				if m := taskPrefix.FindStringSubmatch(s); m != nil {
					s = m[2]
				}
			}
			s = str(ArticleRichText(s, "")["plaintext"])
		}
		if s != "" {
			plain = append(plain, s)
		}
	}
	result := NativeContent{TextContent: strings.Join(plain, "\n\n")}
	host := in.Host
	if host == "markpub" || host == "unknown" {
		var err error
		md := assetURL.ReplaceAllStringFunc(in.Markdown, func(u string) string {
			a, e := resolve(u)
			if e != nil {
				err = e
				return u
			}
			if a == nil {
				err = fmt.Errorf("An article image has not been uploaded.")
				return u
			}
			return a.URL
		})
		if err != nil {
			return NativeContent{}, err
		}
		md = embedURL.ReplaceAllString(md, "[$1]($1)")
		result.Content = Record{"$type": "at.markpub.markdown", "flavor": "gfm", "text": Record{"$type": "at.markpub.text", "markdown": md}}
		return result, nil
	}
	ns := "blog.pckt"
	if host == "leaflet" {
		ns = "pub.leaflet"
	} else if host == "offprint" {
		ns = "app.offprint"
	}
	bn := ns + ".block"
	if host == "leaflet" {
		bn = ns + ".blocks"
	}
	rich := func(s string) Record { return ArticleRichText(s, ns+".richtext.facet") }
	textBlock := func(s string) Record { r := rich(s); r["$type"] = bn + ".text"; return r }
	convert := func(b MarkdownBlock) (Record, error) {
		switch b.Kind {
		case "heading":
			r := rich(blockHeadingText(b))
			t := "heading"
			if host == "leaflet" {
				t = "header"
			}
			level := b.HeadingLevel
			if host == "offprint" && level > 3 {
				level = 3
			}
			r["$type"] = bn + "." + t
			r["level"] = level
			return r, nil
		case "quote":
			r := Record{"$type": bn + ".blockquote"}
			lines := quoteLines(b.Source)
			if host == "leaflet" {
				for k, v := range rich(strings.Join(lines, "\n")) {
					r[k] = v
				}
			} else {
				a := []any{}
				for _, s := range lines {
					a = append(a, textBlock(s))
				}
				r["content"] = a
			}
			return r, nil
		case "code":
			t, f := "codeBlock", "plaintext"
			if host == "leaflet" {
				t = "code"
			}
			if host == "offprint" {
				f = "code"
			}
			r := Record{"$type": bn + "." + t, f: blockCodeText(b)}
			language := strings.ToLower(strings.TrimSpace(b.Language))
			for len(language) > 50 {
				_, size := utf8.DecodeLastRuneInString(language)
				language = language[:len(language)-size]
			}
			if language != "" {
				r["language"] = language
			}
			return r, nil
		case "image":
			a, e := resolve(b.URL)
			if e != nil {
				return nil, e
			}
			if a == nil {
				return nil, fmt.Errorf("Upload remote images before publishing native article content.")
			}
			alt := b.Alt
			if alt == "" {
				alt = a.Alt
			}
			ratio := Record{"width": a.Width, "height": a.Height}
			if host == "pckt" {
				return Record{"$type": bn + ".image", "attrs": Record{"src": "blob:" + str(object(a.Blob["ref"])["$link"]), "blob": a.Blob, "alt": alt, "align": "center", "aspectRatio": ratio}}, nil
			}
			r := Record{"$type": bn + ".image", "image": a.Blob, "alt": alt, "aspectRatio": ratio}
			if host == "offprint" {
				r["width"] = "100%"
				r["alignment"] = "center"
			}
			return r, nil
		case "embed":
			t, f := "website", "src"
			if host == "offprint" {
				t, f = "webEmbed", "href"
			}
			r := Record{"$type": bn + "." + t, f: b.URL, "title": b.URL}
			if host == "offprint" {
				r["width"] = "100%"
				r["alignment"] = "center"
			}
			return r, nil
		case "thematic-break":
			return Record{"$type": bn + ".horizontalRule"}, nil
		}
		return textBlock(b.Source), nil
	}
	type group struct {
		level    int
		list     Record
		children []any
		last     Record
		ordered  bool
	}
	items := []any{}
	stack := []*group{}
	for _, b := range in.Blocks {
		if b.Kind != "ordered-list" && b.Kind != "unordered-list" {
			stack = nil
			r, e := convert(b)
			if e != nil {
				return NativeContent{}, e
			}
			items = append(items, r)
			continue
		}
		ordered := b.Kind == "ordered-list"
		s := listPrefix.ReplaceAllString(b.Source, "")
		task := taskPrefix.FindStringSubmatch(s)
		taskMode := task != nil && host != "leaflet"
		for len(stack) > 0 && stack[len(stack)-1].level > b.ListLevel {
			stack = stack[:len(stack)-1]
		}
		var g *group
		if len(stack) > 0 {
			g = stack[len(stack)-1]
		}
		typ := "bulletList"
		if host == "leaflet" {
			typ = "unorderedList"
		}
		if ordered {
			typ = "orderedList"
		}
		if taskMode {
			typ = "taskList"
		}
		field := "children"
		if host == "pckt" {
			field = "content"
		}
		if g == nil || g.level != b.ListLevel || g.ordered != ordered || g.list["$type"] != bn+"."+typ {
			list := Record{"$type": bn + "." + typ, field: []any{}}
			if ordered && b.ListStart != 0 && b.ListStart != 1 {
				f := "start"
				if host == "leaflet" {
					f = "startIndex"
				}
				list[f] = b.ListStart
			}
			ng := &group{level: b.ListLevel, list: list, children: []any{}, ordered: ordered}
			if g != nil && g.last != nil && b.ListLevel > g.level {
				if host == "pckt" {
					g.last["content"] = append(g.last["content"].([]any), list)
				} else if host == "leaflet" && g.ordered != ordered {
					f := "unorderedListChildren"
					if ordered {
						f = "orderedListChildren"
					}
					g.last[f] = list
				} else {
					g.last["children"] = ng.children
				} // update parent slice when appending below
			} else {
				items = append(items, list)
			}
			g = ng
			stack = append(stack, g)
		}
		if task != nil {
			s = task[2]
		}
		item := Record{"content": textBlock(s)}
		if host == "pckt" {
			t := "listItem"
			if taskMode {
				t = "taskItem"
			}
			item = Record{"$type": bn + "." + t, "content": []any{textBlock(s)}}
		}
		if task != nil {
			item["checked"] = strings.EqualFold(task[1], "x")
		}
		g.children = append(g.children, item)
		g.list[field] = g.children
		g.last = item
		if len(stack) > 1 {
			parent := stack[len(stack)-2]
			if b.ListLevel > parent.level && host != "pckt" && !(host == "leaflet" && parent.ordered != ordered) {
				parent.last["children"] = g.children
			}
		}
	}
	if host == "leaflet" {
		blocks := []any{}
		for _, b := range items {
			blocks = append(blocks, Record{"$type": "pub.leaflet.pages.linearDocument#block", "block": b})
		}
		result.Content = Record{"$type": "pub.leaflet.content", "pages": []any{Record{"$type": "pub.leaflet.pages.linearDocument", "blocks": blocks}}}
		return result, nil
	}
	if host == "pckt" {
		description := strings.TrimSpace(in.Description)
		first := ""
		if len(items) > 0 {
			first = str(object(items[0])["plaintext"])
		}
		if description != "" && description != first {
			r := rich(description)
			r["$type"] = bn + ".heading"
			r["level"] = 3
			items = append([]any{r}, items...)
		}
		if len(items) == 0 || object(items[len(items)-1])["$type"] != bn+".text" {
			items = append(items, textBlock(""))
		}
	}
	result.Content = Record{"$type": ns + ".content", "items": items}
	return result, nil
}
