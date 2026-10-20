package anypub

import (
	"fmt"
	"strings"
	"unicode"
)

func escapeMarkdown(s string) string {
	var b strings.Builder
	for _, r := range s {
		if strings.ContainsRune("\\`*_{}[]()#+!<>|~&=.-", r) {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}
func AppendLiveArticleContent(host string, existing Record, timestamp, text string) (Record, error) {
	addition := Record{}
	switch host {
	case "markpub", "unknown":
		addition = Record{"$type": "at.markpub.markdown", "flavor": "gfm", "text": Record{"$type": "at.markpub.text", "markdown": "## " + timestamp + "\n\n" + escapeMarkdown(text)}}
	case "leaflet":
		addition = Record{"$type": "pub.leaflet.content", "pages": []any{Record{"$type": "pub.leaflet.pages.linearDocument", "blocks": []any{Record{"$type": "pub.leaflet.pages.linearDocument#block", "block": Record{"$type": "pub.leaflet.blocks.header", "level": 2, "plaintext": timestamp}}, Record{"$type": "pub.leaflet.pages.linearDocument#block", "block": Record{"$type": "pub.leaflet.blocks.text", "plaintext": text}}}}}}
	default:
		ns := "blog.pckt"
		if host == "offprint" {
			ns = "app.offprint"
		}
		addition = Record{"$type": ns + ".content", "items": []any{Record{"$type": ns + ".block.heading", "level": 2, "plaintext": timestamp}, Record{"$type": ns + ".block.text", "plaintext": text}}}
	}
	if existing == nil {
		return addition, nil
	}
	if existing["$type"] != addition["$type"] {
		return nil, fmt.Errorf("The article's native format changed. Refresh before adding an update.")
	}
	r := clone(existing)
	switch host {
	case "markpub", "unknown":
		old := object(existing["text"])
		md, ok := old["markdown"].(string)
		if !ok || old["textBlob"] != nil {
			return nil, fmt.Errorf("Blob-backed article content cannot be appended safely.")
		}
		t := clone(old)
		t["markdown"] = md + "\n\n" + str(object(addition["text"])["markdown"])
		r["text"] = t
	case "leaflet":
		pages, ok := array(existing["pages"])
		if !ok || existing["blobPages"] != nil {
			return nil, fmt.Errorf("Blob-backed article pages cannot be appended safely.")
		}
		if len(pages) == 0 {
			return nil, fmt.Errorf("This article has no appendable native page.")
		}
		last := object(pages[len(pages)-1])
		blocks, ok := array(last["blocks"])
		if last["$type"] != "pub.leaflet.pages.linearDocument" || !ok {
			return nil, fmt.Errorf("This article has no appendable native page.")
		}
		next := clone(last)
		addPages, _ := array(addition["pages"])
		addBlocks, _ := array(object(addPages[0])["blocks"])
		next["blocks"] = append(append([]any{}, blocks...), addBlocks...)
		newPages := append([]any{}, pages...)
		newPages[len(newPages)-1] = next
		r["pages"] = newPages
	default:
		items, ok := array(existing["items"])
		if existing["blob"] != nil || !ok {
			return nil, fmt.Errorf("Blob-backed article content cannot be appended safely.")
		}
		more, _ := array(addition["items"])
		r["items"] = append(append([]any{}, items...), more...)
	}
	return r, nil
}
func LivePostText(text string, tags []string) string {
	a := []string{}
	for _, tag := range tags {
		tag = strings.TrimPrefix(tag, "#")
		tag = strings.Map(func(r rune) rune {
			if unicode.IsSpace(r) {
				return -1
			}
			return r
		}, tag)
		a = append(a, "#"+tag)
	}
	parts := []string{}
	if s := strings.TrimSpace(text); s != "" {
		parts = append(parts, s)
	}
	if len(a) > 0 {
		parts = append(parts, strings.Join(a, " "))
	}
	return strings.Join(parts, "\n\n")
}
