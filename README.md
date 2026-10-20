# AnyPubKit

Shared Standard.site record construction for AnyPub and The Social Wire, with Swift and TypeScript entry points in one repository.

The kit builds `site.standard.publication` and `site.standard.document` records, preserves extension fields when editing decoded records, and prepares native Leaflet, MarkPub, Offprint, and pckt content. Apps supply publication identity, timestamps, Markdown parsing, and uploaded blob references. OAuth, PDS requests, concurrency checks, retries, scheduling, and persistence remain in each app.

## Swift

Use Swift 6 or later, on macOS 14+, iOS 17+, or Linux. Pin the repository to a reviewed commit:

```swift
.package(url: "https://github.com/Stygian-Tech/anypub-kit.git", revision: "<commit>")
```

Add `.product(name: "AnyPubKit", package: "anypub-kit")` to the target, then:

```swift
import AnyPubKit

let publication = try StandardSiteRecordBuilder().publicationRecord(
  url: "https://amoreeffectivebutton.blog",
  name: "A More Effective Button"
)
let document = try StandardSiteRecordBuilder().documentRecord(
  site: publicationURI,
  title: title,
  publishedAt: publicationDate,
  path: "/posts/example",
  content: .object([
    "$type": .string("at.markpub.markdown"),
    "flavor": .string("gfm"),
    "text": .object([
      "$type": .string("at.markpub.text"),
      "markdown": .string(markdown)
    ])
  ])
)
```

`updatingDocument(_:updatedAt:changes:)` retains the original publication date and unknown top-level fields. `PublicationContentAdapter.prepare(document:host:description:images:)` converts canonical blocks and returns any content offload separately; upload it through the app's transport, then call `replacingOffload(with:)`.

## TypeScript

Install from the same reviewed commit using a Git dependency:

```json
{
  "dependencies": {
    "@stygian/anypub-kit": "github:Stygian-Tech/anypub-kit#<commit>"
  }
}
```

The package exports TypeScript source for Bun and bundlers; it has no runtime dependencies.

```ts
import { buildPublicationRecord, buildDocumentRecord } from '@stygian/anypub-kit';

const publication = buildPublicationRecord({
  url: 'https://amoreeffectivebutton.blog',
  name: 'A More Effective Button',
});
const document = buildDocumentRecord({
  site: publicationURI,
  title,
  publishedAt: publicationDate.toISOString(),
  path: '/posts/example',
  content,
}, previousRecord);
```

Builders preserve previous extension fields. An omitted property retains its prior value; an explicit `undefined` removes an optional property. Callers retain the original `publishedAt` when updating a document. `buildArticleNativeContent` accepts the app's structural Markdown model, while `appendLiveArticleContent` supports incremental live posts. `prepareContentOffload` and `replaceContentOffload` separate payload preparation from uploading. `buildNativeDocumentWrapper` constructs Offprint/pckt references after the canonical document CID is known.

## Validation and tests

Record builders validate metadata byte and grapheme limits, publication URLs, document sites, safe relative paths, and content discriminators. Content is an open union selected by `$type`; Standard.site does not mandate a single article-body format. Host selection and publication ownership are application policy.

```sh
swift test --enable-swift-testing --disable-xctest --warnings-as-errors
swift build -c release --warnings-as-errors
bun install --frozen-lockfile
bun run typecheck
bun test
```

Both language test suites consume `Fixtures/standard-site.json`. Keep those fixtures aligned when changing the wire contract. CI also checks Swift on Linux and macOS.

Development and consumer adoption are tracked in [ANY-3](https://linear.app/stygian-tech/issue/ANY-3/extract-anypubkit-for-shared-standardsite-record-creation).
