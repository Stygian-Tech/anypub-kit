// Package xrpc exposes stateless publishing preparation through a Lexicon procedure.
// It never receives account tokens and never reads or writes a PDS.
package xrpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	kit "github.com/stygian-tech/anypub-kit"
)

const NSID = "tech.stygian.anypub.prepare"
const MaxRequestBytes int64 = 8 << 20

// Request describes one deterministic operation. Fields not used by an operation
// are ignored; browser credentials and PDS transports are deliberately absent.
type Request struct {
	Operation   string                  `json:"operation"`
	Input       kit.Record              `json:"input,omitempty"`
	Previous    kit.Record              `json:"previous,omitempty"`
	ClearFields []string                `json:"clearFields,omitempty"`
	Native      *kit.NativeContentInput `json:"native,omitempty"`
	Host        string                  `json:"host,omitempty"`
	Content     kit.Record              `json:"content,omitempty"`
	Prepared    *kit.PreparedContent    `json:"prepared,omitempty"`
	Blob        kit.Record              `json:"blob,omitempty"`
	Reference   kit.Record              `json:"reference,omitempty"`
	Site        string                  `json:"site,omitempty"`
	Timestamp   string                  `json:"timestamp,omitempty"`
	Text        string                  `json:"text,omitempty"`
}

type Response struct {
	Record   kit.Record           `json:"record,omitempty"`
	Native   *kit.NativeContent   `json:"native,omitempty"`
	Prepared *kit.PreparedContent `json:"prepared,omitempty"`
	Content  kit.Record           `json:"content,omitempty"`
}

// Prepare is also usable in-process by a Go backend without an HTTP round trip.
func Prepare(request Request) (Response, error) {
	previous := make(kit.Record, len(request.Previous))
	for key, value := range request.Previous {
		previous[key] = value
	}
	for _, key := range request.ClearFields {
		delete(previous, key)
	}
	switch request.Operation {
	case "document":
		record, err := kit.BuildDocumentRecord(request.Input, previous)
		return Response{Record: record}, err
	case "publication":
		record, err := kit.BuildPublicationRecord(request.Input, previous)
		return Response{Record: record}, err
	case "article":
		if request.Native == nil {
			return Response{}, errors.New("native input is required")
		}
		native, err := kit.BuildArticleNativeContent(*request.Native)
		return Response{Native: &native}, err
	case "offload":
		prepared, err := kit.PrepareContentOffload(request.Host, request.Content)
		return Response{Prepared: &prepared}, err
	case "replaceOffload":
		if request.Prepared == nil {
			return Response{}, errors.New("prepared content is required")
		}
		content, err := kit.ReplaceContentOffload(*request.Prepared, request.Blob)
		return Response{Content: content}, err
	case "live":
		content, err := kit.AppendLiveArticleContent(request.Host, request.Content, request.Timestamp, request.Text)
		return Response{Content: content}, err
	case "wrapper":
		record, err := kit.BuildNativeDocumentWrapper(request.Host, request.Reference, request.Site, previous)
		return Response{Record: record}, err
	default:
		return Response{}, fmt.Errorf("unknown preparation operation: %s", request.Operation)
	}
}

// Handler returns a procedure handler suitable for mounting at /xrpc/ + NSID.
func Handler() http.Handler { return http.HandlerFunc(serve) }

func serve(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		fail(w, http.StatusMethodNotAllowed, "InvalidRequest", "Use POST")
		return
	}
	if strings.Split(r.Header.Get("Content-Type"), ";")[0] != "application/json" {
		fail(w, http.StatusUnsupportedMediaType, "InvalidRequest", "Expected application/json")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.UseNumber()
	var request Request
	if err := decoder.Decode(&request); err != nil {
		badJSON(w, err)
		return
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		badJSON(w, err)
		return
	}
	response, err := Prepare(request)
	if err != nil {
		fail(w, http.StatusBadRequest, "InvalidRequest", err.Error())
		return
	}
	// Marshal before sending headers so non-JSON results cannot yield partial success.
	data, err := json.Marshal(response)
	if err != nil {
		fail(w, http.StatusInternalServerError, "InternalServerError", "Could not encode prepared content")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
func badJSON(w http.ResponseWriter, err error) {
	var limit *http.MaxBytesError
	if errors.As(err, &limit) {
		fail(w, http.StatusRequestEntityTooLarge, "InvalidRequest", "Preparation input exceeds 8 MiB")
		return
	}
	fail(w, http.StatusBadRequest, "InvalidRequest", "Expected one valid JSON object")
}
func fail(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": code, "message": message})
}
