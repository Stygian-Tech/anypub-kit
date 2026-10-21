// Package anypub provides deterministic publication records and native content.
// Callers own parsing, timestamps, blob uploads, and persistence.
package anypub

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/rivo/uniseg"
)

type Record = map[string]any

func clone(r Record) Record {
	out := Record{}
	for k, v := range r {
		out[k] = v
	}
	return out
}
func object(v any) Record {
	if r, ok := v.(map[string]any); ok {
		return r
	}
	return nil
}
func str(v any) string { s, _ := v.(string); return s }
func merge(input, previous Record, t string) Record {
	r := clone(previous)
	for k, v := range input {
		r[k] = v
	}
	r["$type"] = t
	return r
}
func fail(field, reason string) error { return fmt.Errorf("%s: %s", field, reason) }
func text(v any, field string, bytes, graphemes int) error {
	s, ok := v.(string)
	if !ok {
		return fail(field, "Expected a string")
	}
	if bytes > 0 && len(s) > bytes {
		return fail(field, fmt.Sprintf("Exceeds %d UTF-8 bytes", bytes))
	}
	if graphemes > 0 && uniseg.GraphemeClusterCount(s) > graphemes {
		return fail(field, fmt.Sprintf("Exceeds %d graphemes", graphemes))
	}
	return nil
}
func clean(s string) bool {
	return !strings.ContainsFunc(s, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) || unicode.Is(unicode.Cf, r) })
}
func validHTTPS(s string) bool {
	if !clean(s) || !strings.HasPrefix(strings.ToLower(s), "https://") || strings.ContainsAny(s, "?#\\") {
		return false
	}
	u, e := url.Parse(s)
	if e != nil || u.Hostname() == "" || u.User != nil || strings.Contains(strings.SplitN(s[8:], "/", 2)[0], "@") {
		return false
	}
	if p := u.Port(); p != "" {
		n, err := strconv.Atoi(p)
		if err != nil || n > 65535 {
			return false
		}
	}
	if strings.Contains(u.Host, "[") && net.ParseIP(u.Hostname()) == nil {
		return false
	}
	return true
}

var publicationPattern = regexp.MustCompile(`^at://([^/]+)/site\.standard\.publication/([A-Za-z0-9._~:-]+)$`)
var didPattern = regexp.MustCompile(`^did:[a-z]+:([A-Za-z0-9._:-]|%[0-9A-Fa-f]{2})+$`)
var handlePattern = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9.-]*[A-Za-z0-9])?\.[A-Za-z]{2,}$`)

func validPublication(s string) bool {
	m := publicationPattern.FindStringSubmatch(s)
	if len(m) == 0 || !clean(s) || strings.ContainsAny(s, "?#@") {
		return false
	}
	return (didPattern.MatchString(m[1]) || (!strings.HasPrefix(m[1], "did:") && handlePattern.MatchString(m[1]))) && m[2] != "." && m[2] != ".." && len(m[2]) <= 512
}
func validPath(s string) bool {
	if s == "" || !clean(s) || strings.HasPrefix(s, "//") {
		return false
	}
	d, e := url.PathUnescape(s)
	if e != nil || !clean(d) || strings.HasPrefix(d, "//") || strings.ContainsAny(d, "\\?#:%") {
		return false
	}
	for _, p := range strings.Split(d, "/") {
		if p == "." || p == ".." {
			return false
		}
	}
	return true
}

var nsPart = regexp.MustCompile(`^[A-Za-z]([A-Za-z0-9-]*[A-Za-z0-9])?$`)
var nsLast = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]*$`)
var fragment = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)

func validType(s string) bool {
	pieces := strings.Split(s, "#")
	parts := strings.Split(pieces[0], ".")
	if len(pieces) > 2 || len(parts) < 3 || len(pieces[0]) > 317 {
		return false
	}
	if len(pieces) == 2 && (pieces[1] == "main" || !fragment.MatchString(pieces[1])) {
		return false
	}
	for i, p := range parts {
		pattern := nsPart
		if i == len(parts)-1 {
			pattern = nsLast
		}
		if len(p) > 63 || !pattern.MatchString(p) {
			return false
		}
	}
	return true
}

var timestampPattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d{1,9})?(Z|[+-]\d{2}:\d{2})$`)

func validateTimestamp(v any, field string) error {
	if e := text(v, field, 0, 0); e != nil {
		return e
	}
	s := str(v)
	if !timestampPattern.MatchString(s) {
		return fail(field, "Expected an ISO timestamp with a timezone")
	}
	_, e := time.Parse(time.RFC3339Nano, s)
	if e != nil {
		return fail(field, "Invalid calendar date or time")
	}
	z := s[len(s)-6:]
	if z[0] == '+' || z[0] == '-' {
		if z[1:3] > "23" || z[4:6] > "59" {
			return fail(field, "Invalid calendar date or time")
		}
	}
	return nil
}

// BuildDocumentRecord merges input into previous without mutating either record.
func BuildDocumentRecord(input Record, previous Record) (Record, error) {
	r := merge(input, previous, "site.standard.document")
	if e := text(r["site"], "site", 0, 0); e != nil {
		return nil, e
	}
	if s := str(r["site"]); !validHTTPS(s) && !validPublication(s) {
		return nil, fail("site", "Expected an AT publication URI or HTTPS URL")
	}
	if e := text(r["title"], "title", 5000, 500); e != nil {
		return nil, e
	}
	if e := validateTimestamp(r["publishedAt"], "publishedAt"); e != nil {
		return nil, e
	}
	if v, ok := r["updatedAt"]; ok {
		if e := validateTimestamp(v, "updatedAt"); e != nil {
			return nil, e
		}
	}
	for _, f := range []string{"description", "textContent", "path"} {
		if v, ok := r[f]; ok {
			b, g := 0, 0
			if f == "description" {
				b, g = 30000, 3000
			}
			if e := text(v, f, b, g); e != nil {
				return nil, e
			}
			if f == "path" && !validPath(str(v)) {
				return nil, fail(f, "Expected a relative path without traversal, query, fragment, or authority")
			}
		}
	}
	for _, f := range []string{"tags", "langs"} {
		if v, ok := r[f]; ok {
			var a []any
			switch x := v.(type) {
			case []any:
				a = x
			case []string:
				for _, s := range x {
					a = append(a, s)
				}
			default:
				return nil, fail(f, "Expected an array of strings")
			}
			for _, s := range a {
				b, g := 0, 0
				if f == "tags" {
					b, g = 1280, 128
				}
				if e := text(s, f, b, g); e != nil {
					return nil, e
				}
			}
		}
	}
	if v, ok := r["content"]; ok {
		if !validType(str(object(v)["$type"])) {
			return nil, fail("content", "Expected an object with a valid Lexicon $type")
		}
	}
	return r, nil
}
func BuildPublicationRecord(input Record, previous Record) (Record, error) {
	r := merge(input, previous, "site.standard.publication")
	if e := text(r["url"], "url", 0, 0); e != nil {
		return nil, e
	}
	if !validHTTPS(str(r["url"])) {
		return nil, fail("url", "Expected an HTTPS publication URL")
	}
	r["url"] = strings.TrimRight(str(r["url"]), "/")
	if e := text(r["name"], "name", 5000, 500); e != nil {
		return nil, e
	}
	if v, ok := r["description"]; ok {
		if e := text(v, "description", 30000, 3000); e != nil {
			return nil, e
		}
	}
	return r, nil
}
func BuildNativeDocumentWrapper(host string, reference Record, site string, previous Record) (Record, error) {
	if host != "offprint" && host != "pckt" {
		return nil, fail("host", "Expected offprint or pckt")
	}
	for _, f := range []string{"uri", "cid"} {
		if e := text(reference[f], "document."+f, 0, 0); e != nil {
			return nil, e
		}
		if str(reference[f]) == "" {
			return nil, fail("document", "Expected a URI and CID")
		}
	}
	doc := clone(object(previous["document"]))
	for k, v := range reference {
		doc[k] = v
	}
	typ := "app.offprint.document.article"
	if host == "pckt" {
		typ = "blog.pckt.document"
	}
	r := merge(Record{"document": doc}, previous, typ)
	if host == "offprint" {
		doc["$type"] = "com.atproto.repo.strongRef"
	} else {
		delete(doc, "$type")
		if site != "" {
			r["site"] = site
		}
		if str(r["site"]) == "" || !clean(str(r["site"])) {
			return nil, fail("site", "Pckt requires a site")
		}
	}
	return r, nil
}
