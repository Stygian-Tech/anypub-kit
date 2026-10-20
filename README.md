# AnyPubKit

One Go implementation of Standard.site record and native-content preparation, used in-process by AnyPub and through XRPC by The Social Wire. The TypeScript package is an asynchronous HTTP client; publishing rules live in Go.

## Go

```sh
go get github.com/stygian-tech/anypub-kit@<reviewed-commit>
```

```go
import anypub "github.com/stygian-tech/anypub-kit"

publication, err := anypub.BuildPublicationRecord(anypub.Record{
    "url": "https://amoreeffectivebutton.blog",
    "name": "A More Effective Button",
}, nil)
```

`BuildDocumentRecord(input, previous)` and `BuildPublicationRecord(input, previous)` retain unknown previous fields and apply explicit input changes. Callers supply lifecycle timestamps and retain `publishedAt` when editing. `BuildNativeDocumentWrapper` constructs Offprint/pckt references after the canonical CID is known.

`BuildArticleNativeContent` accepts normalized Markdown blocks from the caller's parser. `PrepareContentOffload` returns the exact UTF-8 body for upload; pass the uploaded blob to `ReplaceContentOffload`. `AppendLiveArticleContent` preserves existing native blocks and rejects unsupported blob-backed append operations. Leaflet, MarkPub, Offprint, and pckt formats are supported.

The kit has no account storage, token handling, clock reads, or PDS transport. AnyPub owns its backend's OAuth, persistence, uploads, and scheduled publishing. Social Wire retains browser OAuth, PDS writes, stable keys, CAS checks, and uncertain-write reconciliation.

## Go service

Mount `xrpc.Handler()` at `/xrpc/` + `xrpc.NSID` in a Go server, or run:

```sh
go run ./cmd/anypubkit
```

`PORT` defaults to `8080`. `GET /health` reports readiness. The Lexicon procedure is `POST /xrpc/tech.stygian.anypub.prepare`; its schema is in `lexicons/`. Operations are `publication`, `document`, `article`, `offload`, `replaceOffload`, `live`, and `wrapper`. Requests are limited to 8 MiB and responses use `Cache-Control: no-store`. It receives structured content only and performs no network requests or PDS writes.

## Browser transport

Pin the same reviewed Git revision:

```json
{"dependencies":{"@stygian/anypub-kit":"github:Stygian-Tech/anypub-kit#<commit>"}}
```

```ts
import { buildPublicationRecord } from '@stygian/anypub-kit';
const publication = await buildPublicationRecord({
  url: 'https://amoreeffectivebutton.blog',
  name: 'A More Effective Button',
});
```

All preparation functions return promises. The default endpoint is the same-origin `/xrpc/tech.stygian.anypub.prepare`; options accept `endpoint`, `signal`, and `fetcher`. Calls omit browser credentials. Social Wire's proxy forwards to its server-only `ANYPUB_API_URL`, pointing to the Go AnyPub backend. Optional properties set explicitly to `undefined` are encoded as `clearFields`, retaining editor clearing behavior across JSON.

The package exports TypeScript source for Bun/bundlers, with no runtime dependencies. Next.js consumers must include `@stygian/anypub-kit` in `transpilePackages`. There is no local record-construction fallback when the service is unavailable.

## Checks

```sh
go test -race ./...
go vet ./...
go build ./...
bun install --frozen-lockfile
bun run typecheck
bun test
```

CI runs Go on Linux using the version in `go.mod`, plus browser-client typechecks and tests. `Fixtures/standard-site.json` and `Fixtures/native-content.json` lock record and native-content compatibility with the previous implementations.

Tracked in [ANY-3](https://linear.app/stygian-tech/issue/ANY-3/extract-anypubkit-for-shared-standardsite-record-creation).
