package anypub

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf16"
)

type ContentOffload struct {
	Kind     string `json:"kind"`
	MIMEType string `json:"mimeType"`
	Body     string `json:"body"`
}
type PreparedContent struct {
	Content Record          `json:"content"`
	Offload *ContentOffload `json:"offload,omitempty"`
}

// wireJSON emits compact UTF-8 JSON without HTML escaping, matching browser bodies.
func wireJSON(v any) (string, error) {
	var b bytes.Buffer
	e := json.NewEncoder(&b)
	e.SetEscapeHTML(false)
	if err := e.Encode(v); err != nil {
		return "", err
	}
	return strings.NewReplacer(`\u2028`, " ", `\u2029`, " ").Replace(strings.TrimSuffix(b.String(), "\n")), nil
}
func array(v any) ([]any, bool) {
	switch a := v.(type) {
	case []any:
		return a, true
	case []Record:
		r := make([]any, len(a))
		for i := range a {
			r[i] = a[i]
		}
		return r, true
	}
	return nil, false
}
func PrepareContentOffload(host string, content Record) (PreparedContent, error) {
	p := PreparedContent{Content: content}
	var body, kind, mime string
	limit := 0
	switch host {
	case "markpub", "unknown":
		var ok bool
		body, ok = object(content["text"])["markdown"].(string)
		if !ok {
			return p, fmt.Errorf("Invalid Markpub Markdown content.")
		}
		kind, mime, limit = "markpub-markdown", "text/markdown", 100*1024
	case "leaflet", "pckt":
		field := "pages"
		kind, limit = "leaflet-pages", 100*1024
		if host == "pckt" {
			field, kind, limit = "items", "pckt-items", 20000
		}
		if _, ok := array(content[field]); !ok {
			return p, fmt.Errorf("Invalid %s %s.", host, field)
		}
		var e error
		body, e = wireJSON(content[field])
		if e != nil {
			return p, e
		}
		mime = "application/json"
	default:
		body, e := wireJSON(content)
		if e != nil {
			return p, e
		}
		if len(body) >= 900000 {
			return p, fmt.Errorf("Offprint article content exceeds its record size limit.")
		}
		return p, nil
	}
	if len(body) > limit {
		p.Offload = &ContentOffload{Kind: kind, MIMEType: mime, Body: body}
	}
	return p, nil
}
func numeric(v any) float64 {
	switch n := v.(type) {
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case float64:
		return n
	case json.Number:
		f, _ := n.Float64()
		return f
	}
	return -1
}

// utf16Prefix preserves the browser's 1000 UTF-16 unit preview boundary.
func utf16Prefix(s string, n int) string {
	u := utf16.Encode([]rune(s))
	if len(u) > n {
		u = u[:n]
	}
	return string(utf16.Decode(u))
}
func ReplaceContentOffload(p PreparedContent, blob Record) (Record, error) {
	if p.Offload == nil {
		return p.Content, nil
	}
	o := p.Offload
	if blob["$type"] != "blob" || str(object(blob["ref"])["$link"]) == "" || blob["mimeType"] != o.MIMEType || numeric(blob["size"]) != float64(len(o.Body)) {
		return nil, fmt.Errorf("The content blob does not match the offload plan.")
	}
	r := clone(p.Content)
	switch o.Kind {
	case "markpub-markdown":
		t := clone(object(r["text"]))
		t["textBlob"] = blob
		t["markdown"] = utf16Prefix(o.Body, 1000)
		r["text"] = t
	case "leaflet-pages":
		r["blobPages"] = blob
		r["pages"] = []any{}
		r["blobs"] = []any{}
	case "pckt-items":
		r["blob"] = blob
		r["references"] = []any{}
		delete(r, "items")
	default:
		return nil, fmt.Errorf("Unknown content offload kind.")
	}
	return r, nil
}
