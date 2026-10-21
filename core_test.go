package anypub

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

func equalJSON(t *testing.T, a, b any) {
	t.Helper()
	aa, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	var av, bv any
	_ = json.Unmarshal(aa, &av)
	_ = json.Unmarshal(bb, &bv)
	if !reflect.DeepEqual(av, bv) {
		t.Fatalf("got %s\nwant %s", aa, bb)
	}
}
func TestGoldenRecords(t *testing.T) {
	data, e := os.ReadFile("Fixtures/standard-site.json")
	if e != nil {
		t.Fatal(e)
	}
	var fixtures []struct {
		Name, Kind      string
		Input, Expected Record
	}
	if e = json.Unmarshal(data, &fixtures); e != nil {
		t.Fatal(e)
	}
	for _, f := range fixtures {
		t.Run(f.Name, func(t *testing.T) {
			var got Record
			var err error
			if f.Kind == "publication" {
				got, err = BuildPublicationRecord(f.Input, nil)
			} else {
				got, err = BuildDocumentRecord(f.Input, nil)
			}
			if err != nil {
				t.Fatal(err)
			}
			equalJSON(t, got, f.Expected)
		})
	}
}
func document() Record {
	return Record{"site": "https://example.com", "title": "Title", "publishedAt": "2026-10-01T12:30:00.000Z"}
}
func TestRecordValidation(t *testing.T) {
	for _, v := range []string{"at://did:web:example.com%3A443/site.standard.publication/a", "at://example.com/site.standard.publication/a"} {
		r := document()
		r["site"] = v
		if _, e := BuildDocumentRecord(r, nil); e != nil {
			t.Fatal(e)
		}
	}
	for _, tc := range []struct {
		field string
		value any
	}{{"site", "https://example.com/?x"}, {"site", "https://example.com:999999"}, {"site", "https://[]"}, {"title", nil}, {"site", "at://did:web:a%xx/site.standard.publication/a"}, {"path", "/%2e%2e/a"}, {"path", "/%252e"}, {"path", "//evil"}, {"path", "/a%3fb"}, {"title", strings.Repeat("e\u0301", 501)}, {"title", strings.Repeat("a", 5001)}, {"description", strings.Repeat("x", 3001)}, {"tags", []any{strings.Repeat("é", 129)}}, {"publishedAt", "2026-02-30T12:00:00Z"}, {"publishedAt", "2026-10-01T12:00:00+24:00"}, {"content", Record{"$type": "at.markpub.markdown#main"}}, {"content", []any{}}, {"langs", []any{2}}} {
		t.Run(tc.field+str(tc.value), func(t *testing.T) {
			r := document()
			r[tc.field] = tc.value
			if _, e := BuildDocumentRecord(r, nil); e == nil {
				t.Fatal("accepted invalid field")
			}
		})
	}
	r := document()
	r["title"] = strings.Repeat("👨‍👩‍👧‍👦", 201)
	if _, e := BuildDocumentRecord(r, nil); e == nil {
		t.Fatal("metadata byte limit bypassed")
	}
	r = document()
	r["title"] = strings.Repeat("e\u0301", 500)
	if _, e := BuildDocumentRecord(r, nil); e != nil {
		t.Fatal(e)
	}
	if _, e := BuildPublicationRecord(Record{"url": "https://example.com/ /", "name": "a"}, nil); e == nil {
		t.Fatal("normalized invalid URL before checking")
	}
}
func TestRecordExtensionsAndWrapper(t *testing.T) {
	previous := document()
	previous["custom"] = Record{"a": true}
	next, e := BuildDocumentRecord(Record{"title": "New"}, previous)
	if e != nil {
		t.Fatal(e)
	}
	equalJSON(t, next["custom"], previous["custom"])
	if previous["title"] != "Title" {
		t.Fatal("mutated previous")
	}
	prior := Record{"custom": true, "document": Record{"customRef": true, "$type": "old"}, "site": "site"}
	for _, host := range []string{"offprint", "pckt"} {
		r, e := BuildNativeDocumentWrapper(host, Record{"uri": "at://x/y/z", "cid": "cid"}, "", prior)
		if e != nil {
			t.Fatal(e)
		}
		if object(r["document"])["customRef"] != true || r["custom"] != true {
			t.Fatal(r)
		}
		if host == "pckt" && object(r["document"])["$type"] != nil {
			t.Fatal(r)
		}
	}
	if object(prior["document"])["$type"] != "old" {
		t.Fatal("mutated ref")
	}
}
func TestRichText(t *testing.T) {
	equalJSON(t, ArticleRichText("😀 **é** [bad](javascript:alert)", "pub.leaflet.richtext.facet"), Record{"plaintext": "😀 é bad", "facets": []any{Record{"index": Record{"byteStart": 5, "byteEnd": 7}, "features": []any{Record{"$type": "pub.leaflet.richtext.facet#bold"}}}}})
	r := ArticleRichText("**a _é_** `**literal**` [ok](https://example.com)", "x.y.z")
	if r["plaintext"] != "a é **literal** ok" {
		t.Fatal(r)
	}
	f, _ := array(r["facets"])
	if len(f) != 4 {
		t.Fatal(r)
	}
}
func imageAsset() ImageAsset {
	return ImageAsset{ID: "image", Blob: Record{"$type": "blob", "ref": Record{"$link": "bafkreitest"}, "mimeType": "image/png", "size": 3}, Alt: "Photo", Width: 640, Height: 480, URL: "https://pds.test/blob"}
}
func TestNativeProviders(t *testing.T) {
	blocks := []MarkdownBlock{{Kind: "heading", Source: "Title", RawSource: "###### Title", HeadingLevel: 6}, {Kind: "unordered-list", Source: "- [x] Done", ListStart: 1}, {Kind: "ordered-list", Source: "  3. Child", ListLevel: 1, ListStart: 3}, {Kind: "code", Source: "literal **code**", RawSource: "```ts\nliteral **code**\n```", Language: " TS "}, {Kind: "quote", Source: "> One\n> Two"}, {Kind: "image", URL: "article-asset://image", Alt: "Caption"}, {Kind: "embed", URL: "https://example.com"}, {Kind: "thematic-break", Source: "---"}}
	for _, host := range []string{"leaflet", "offprint", "pckt"} {
		t.Run(host, func(t *testing.T) {
			r, e := BuildArticleNativeContent(NativeContentInput{Host: host, Description: " Summary ", Blocks: blocks, Assets: []ImageAsset{imageAsset()}})
			if e != nil {
				t.Fatal(e)
			}
			if !strings.Contains(r.TextContent, "literal **code**\n\nOne\nTwo\n\nCaption") {
				t.Fatal(r.TextContent)
			}
			var items []any
			if host == "leaflet" {
				p, _ := array(r.Content["pages"])
				wrapped, _ := array(object(p[0])["blocks"])
				for _, w := range wrapped {
					items = append(items, object(w)["block"])
				}
			} else {
				items, _ = array(r.Content["items"])
			}
			if host == "pckt" {
				if object(items[0])["plaintext"] != "Summary" || object(items[len(items)-1])["plaintext"] != "" {
					t.Fatal(items)
				}
				items = items[1:]
			}
			level := 6
			if host == "offprint" {
				level = 3
			}
			if object(items[0])["level"] != level {
				t.Fatal(items[0])
			}
			list := object(items[1])
			children, _ := array(list["children"])
			if host == "pckt" {
				children, _ = array(list["content"])
			}
			if len(children) != 1 || object(children[0])["checked"] != true {
				t.Fatal(list)
			}
			if host == "leaflet" && object(object(children[0])["orderedListChildren"])["startIndex"] != 3 {
				t.Fatal(list)
			}
			code := object(items[2])
			if code["language"] != "ts" {
				t.Fatal(code)
			}
			key := "plaintext"
			if host == "offprint" {
				key = "code"
			}
			if code[key] != "literal **code**" {
				t.Fatal(code)
			}
		})
	}
}
func TestListGrouping(t *testing.T) {
	blocks := []MarkdownBlock{{Kind: "unordered-list", Source: "- A"}, {Kind: "unordered-list", Source: "  - B", ListLevel: 1}, {Kind: "unordered-list", Source: "  - C", ListLevel: 1}, {Kind: "unordered-list", Source: "- D"}, {Kind: "paragraph", Source: "reset"}, {Kind: "unordered-list", Source: "- E"}}
	for _, host := range []string{"leaflet", "offprint", "pckt"} {
		r, e := BuildArticleNativeContent(NativeContentInput{Host: host, Blocks: blocks})
		if e != nil {
			t.Fatal(e)
		}
		s, _ := wireJSON(r.Content)
		if !strings.Contains(s, "\"plaintext\":\"C\"") || !strings.Contains(s, "\"plaintext\":\"D\"") {
			t.Fatal(s)
		}
	}
}
func TestMediaAndMarkpub(t *testing.T) {
	for _, tc := range []struct {
		host, url string
		asset     ImageAsset
	}{{"leaflet", "article-asset://missing", imageAsset()}, {"pckt", "https://example.com/a.png", imageAsset()}, {"markpub", "javascript:bad", imageAsset()}, {"offprint", "article-asset://image", ImageAsset{ID: "image"}}} {
		if _, e := BuildArticleNativeContent(NativeContentInput{Host: tc.host, Assets: []ImageAsset{tc.asset}, Blocks: []MarkdownBlock{{Kind: "image", URL: tc.url}}}); e == nil {
			t.Fatal(tc)
		}
	}
	r, e := BuildArticleNativeContent(NativeContentInput{Host: "markpub", Markdown: "![x](article-asset://image)\n@[embed](https://example.com)", Assets: []ImageAsset{imageAsset()}, Blocks: []MarkdownBlock{{Kind: "image", URL: "article-asset://image", Alt: "x"}}})
	if e != nil {
		t.Fatal(e)
	}
	if object(r.Content["text"])["markdown"] != "![x](https://pds.test/blob)\n[https://example.com](https://example.com)" {
		t.Fatal(r)
	}
}
func TestOffload(t *testing.T) {
	for _, tc := range []struct {
		host, kind string
		content    Record
	}{{"markpub", "markpub-markdown", Record{"custom": true, "text": Record{"custom": true, "markdown": strings.Repeat("é", 52000)}}}, {"leaflet", "leaflet-pages", Record{"custom": true, "pages": []any{Record{"blocks": []any{strings.Repeat("é", 52000)}}}}}, {"pckt", "pckt-items", Record{"custom": true, "items": []any{strings.Repeat("é", 11000)}}}} {
		t.Run(tc.host, func(t *testing.T) {
			before, _ := wireJSON(tc.content)
			p, e := PrepareContentOffload(tc.host, tc.content)
			if e != nil {
				t.Fatal(e)
			}
			if p.Offload == nil || p.Offload.Kind != tc.kind {
				t.Fatal(p)
			}
			b := Record{"$type": "blob", "ref": Record{"$link": "cid"}, "mimeType": p.Offload.MIMEType, "size": len(p.Offload.Body)}
			r, e := ReplaceContentOffload(p, b)
			if e != nil {
				t.Fatal(e)
			}
			if r["custom"] != true {
				t.Fatal(r)
			}
			if tc.host == "markpub" && object(r["text"])["markdown"] != strings.Repeat("é", 1000) {
				t.Fatal("preview")
			}
			after, _ := wireJSON(tc.content)
			if before != after {
				t.Fatal("mutated")
			}
			b["size"] = 0
			if _, e := ReplaceContentOffload(p, b); e == nil {
				t.Fatal("bad blob accepted")
			}
			j, _ := wireJSON(p)
			if !strings.Contains(j, "\"body\":\"") {
				t.Fatal("body is not string")
			}
		})
	}
	p, e := PrepareContentOffload("markpub", Record{"text": Record{"markdown": strings.Repeat("x", 102400)}})
	if e != nil || p.Offload != nil {
		t.Fatal(p, e)
	}
	if _, e := PrepareContentOffload("offprint", Record{"items": []any{strings.Repeat("x", 900000)}}); e == nil {
		t.Fatal("oversize offprint")
	}
}
func TestLiveAppend(t *testing.T) {
	for _, host := range []string{"leaflet", "offprint", "pckt", "markpub", "unknown"} {
		initial, e := AppendLiveArticleContent(host, nil, "stamp", "Old")
		if e != nil {
			t.Fatal(e)
		}
		initial["custom"] = true
		before, _ := wireJSON(initial)
		next, e := AppendLiveArticleContent(host, initial, "stamp", "**Literal**")
		if e != nil {
			t.Fatal(e)
		}
		if next["custom"] != true {
			t.Fatal(next)
		}
		after, _ := wireJSON(initial)
		if before != after {
			t.Fatal("mutated previous")
		}
		if host == "markpub" || host == "unknown" {
			if !strings.Contains(str(object(next["text"])["markdown"]), "\\*\\*Literal\\*\\*") {
				t.Fatal(next)
			}
		} else {
			s, _ := wireJSON(next)
			if !strings.Contains(s, `"plaintext":"**Literal**"`) {
				t.Fatal(next)
			}
		}
	}
	for _, tc := range []struct {
		host string
		r    Record
	}{{"markpub", Record{"$type": "at.markpub.markdown", "text": Record{"markdown": "old", "textBlob": Record{}}}}, {"leaflet", Record{"$type": "pub.leaflet.content", "pages": []any{}, "blobPages": Record{}}}, {"offprint", Record{"$type": "other"}}} {
		if _, e := AppendLiveArticleContent(tc.host, tc.r, "stamp", "new"); e == nil {
			t.Fatal(tc)
		}
	}
	if LivePostText(" Update ", []string{"#live", "two words"}) != "Update\n\n#live #twowords" {
		t.Fatal("live tags")
	}
}

func TestNativeContentGolden(t *testing.T) {
	data, err := os.ReadFile("Fixtures/native-content.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []struct {
		Name     string
		Input    NativeContentInput
		Expected NativeContent
	}
	if err = json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	if len(fixtures) != 12 {
		t.Fatalf("expected 12 native wire fixtures, got %d", len(fixtures))
	}
	for _, f := range fixtures {
		t.Run(f.Name, func(t *testing.T) {
			got, err := BuildArticleNativeContent(f.Input)
			if err != nil {
				t.Fatal(err)
			}
			equalJSON(t, got, f.Expected)
		})
	}
}

func TestOffloadJSONSeparatorsAndLiteralEscapes(t *testing.T) {
	for _, value := range []string{`\u2028`, `\u2029`, `\\u2028`, "\u2028", "\u2029", "\\\u2028"} {
		original := value + strings.Repeat("x", 20000)
		prepared, err := PrepareContentOffload("pckt", Record{"items": []any{Record{"plaintext": original}}})
		if err != nil {
			t.Fatal(err)
		}
		if prepared.Offload == nil {
			t.Fatal("expected offload")
		}
		var decoded []Record
		if err := json.Unmarshal([]byte(prepared.Offload.Body), &decoded); err != nil {
			t.Fatalf("invalid body: %v", err)
		}
		if decoded[0]["plaintext"] != original {
			t.Fatal("content changed while encoding")
		}
	}
}
func TestMarkpubMaximumBytes(t *testing.T) {
	for _, host := range []string{"markpub", "unknown"} {
		if _, err := PrepareContentOffload(host, Record{"text": Record{"markdown": strings.Repeat("a", 1_000_000)}}); err != nil {
			t.Fatal(err)
		}
		if _, err := PrepareContentOffload(host, Record{"text": Record{"markdown": strings.Repeat("a", 1_000_001)}}); err == nil {
			t.Fatal("accepted oversized Markdown")
		}
	}
}
