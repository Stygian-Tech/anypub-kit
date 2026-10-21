package xrpc_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stygian-tech/anypub-kit/xrpc"
)

func TestPublishedProcedureContract(t *testing.T) {
	data, err := os.ReadFile("../lexicons/at.anypub.prepare.json")
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		ID   string `json:"id"`
		Defs map[string]struct {
			Type string `json:"type"`
		} `json:"defs"`
	}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatal(err)
	}
	if schema.ID != "at.anypub.prepare" || schema.Defs["main"].Type != "procedure" {
		t.Fatalf("published schema and handler disagree: %+v", schema)
	}
	if schema.ID != xrpc.NSID {
		t.Fatalf("handler NSID %q differs from published schema %q", xrpc.NSID, schema.ID)
	}
	mux := http.NewServeMux()
	mux.Handle("/xrpc/"+xrpc.NSID, xrpc.Handler())
	r := httptest.NewRequest(http.MethodPost, "/xrpc/at.anypub.prepare", strings.NewReader(`{"operation":"publication","input":{"url":"https://example.com","name":"Publication"}}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("published procedure: %d %s", w.Code, w.Body)
	}
}

func call(t *testing.T, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/xrpc/"+xrpc.NSID, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	xrpc.Handler().ServeHTTP(recorder, request)
	return recorder
}
func TestPublicationAndDocumentPreparation(t *testing.T) {
	for _, body := range []string{
		`{"operation":"publication","input":{"url":"https://example.com/","name":"Publication"}}`,
		`{"operation":"document","input":{"site":"https://example.com","title":"Title","publishedAt":"2026-10-01T12:00:00.000Z"}}`,
	} {
		response := call(t, body)
		if response.Code != 200 {
			t.Fatalf("prepare: %d %s", response.Code, response.Body)
		}
		var result map[string]any
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		record, ok := result["record"].(map[string]any)
		if !ok || record["$type"] == nil {
			t.Fatalf("missing record: %s", response.Body)
		}
		if response.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("unpublished content must not be cached")
		}
	}
}
func TestClearedFieldsAndUnknownExtensions(t *testing.T) {
	response := call(t, `{"operation":"document","input":{"site":"https://example.com","title":"Edited","publishedAt":"2026-10-01T12:00:00.000Z"},"previous":{"description":"old","future":{"nested":true}},"clearFields":["description"]}`)
	if response.Code != 200 {
		t.Fatalf("prepare: %d %s", response.Code, response.Body)
	}
	var result struct {
		Record map[string]any `json:"record"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if _, ok := result.Record["description"]; ok {
		t.Fatal("explicit clearing was lost in JSON transport")
	}
	if result.Record["future"] == nil {
		t.Fatal("extension was dropped")
	}
}
func TestMalformedAndInvalidOperations(t *testing.T) {
	for _, body := range []string{`null`, `{}`, `{"operation":"unknown"}`, `{} {}`, `{"operation":"article"}`, `{"operation":"replaceOffload"}`, `{"operation":"document","input":{"site":"http://unsafe","title":"x"}}`} {
		response := call(t, body)
		if response.Code != 400 {
			t.Errorf("body %s: %d %s", body, response.Code, response.Body)
		}
		if !json.Valid(response.Body.Bytes()) {
			t.Errorf("not XRPC JSON: %s", response.Body)
		}
	}
}
func TestRequestLimitsAndMethod(t *testing.T) {
	response := call(t, `{"operation":"publication","input":{"name":"`+strings.Repeat("x", int(xrpc.MaxRequestBytes))+`"}}`)
	if response.Code != 413 {
		t.Fatalf("oversize: %d", response.Code)
	}
	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
		recorder := httptest.NewRecorder()
		xrpc.Handler().ServeHTTP(recorder, httptest.NewRequest(method, "/", nil))
		if recorder.Code != 405 || recorder.Header().Get("Allow") != "POST" {
			t.Fatalf("method %s: %d", method, recorder.Code)
		}
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("POST", "/", bytes.NewBufferString("{}"))
	request.Header.Set("Content-Type", "text/plain")
	xrpc.Handler().ServeHTTP(recorder, request)
	if recorder.Code != 415 {
		t.Fatalf("media type: %d", recorder.Code)
	}
}

func TestIntegerExtensionsRemainExact(t *testing.T) {
	response := call(t, `{"operation":"document","input":{"site":"https://example.com","title":"Edited","publishedAt":"2026-10-01T12:00:00Z"},"previous":{"extension":{"integer":9007199254740993}}}`)
	if response.Code != 200 || !strings.Contains(response.Body.String(), `9007199254740993`) {
		t.Fatalf("integer changed: %d %s", response.Code, response.Body)
	}
}
