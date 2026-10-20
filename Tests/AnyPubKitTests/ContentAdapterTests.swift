import AnyPubKit
import Foundation
import Testing

private let blob = ATProtoBlobRef(ref: .init(link: "bafkexample"), mimeType: "image/png", size: 123)
private func doc(_ markdown: String, blocks: [CanonicalBlock] = []) -> CanonicalDocument {
  CanonicalDocument(blocks: blocks, plaintext: "Plain", markdown: markdown)
}

@Test func markpubByteBoundariesAndOffloadPreview() throws {
  let inline = try PublicationContentAdapter.prepare(
    document: doc(String(repeating: "x", count: 102_400)), host: .markpub)
  #expect(inline.offload == nil)
  let offloaded = try PublicationContentAdapter.prepare(
    document: doc(String(repeating: "👋", count: 25_601)), host: .unknown)
  #expect(offloaded.host == .markpub)
  #expect(offloaded.offload?.mimeType == "text/markdown")
  if case .markpubMarkdown(let bytes) = offloaded.offload {
    #expect(bytes.count == 102_404)
  } else {
    Issue.record("Expected Markdown offload")
  }
  let replacement = offloaded.replacingOffload(with: blob).objectValue?["text"]?.objectValue
  #expect(replacement?["markdown"]?.stringValue == String(repeating: "👋", count: 1_000))
  #expect(replacement?["textBlob"]?.blobCID == blob.ref.link)
  _ = try PublicationContentAdapter.prepare(
    document: doc(String(repeating: "a", count: 1_000_000)), host: .markpub)
  #expect(throws: PublicationContentError.self) {
    try PublicationContentAdapter.prepare(
      document: doc(String(repeating: "a", count: 1_000_001)), host: .markpub)
  }
  #expect(inline.replacingOffload(with: blob) == inline.content)
}

@Test func markpubAssetAndEmbedReplacement() throws {
  let id = UUID(uuidString: "00000000-0000-0000-0000-000000000001")!
  let image = PublishedBodyImage(
    assetID: id, blob: blob, alt: "ignored", width: 5, height: 3,
    publicURL: "https://images.example/a.png")
  let result = try PublicationContentAdapter.prepare(
    document: doc(
      "![alt](anypub-asset://\(id.uuidString))\n@[embed](https://example.com)",
      blocks: [.image(assetID: id, alt: "alt")]), host: .markpub, images: [image])
  #expect(
    result.content.objectValue?["text"]?.objectValue?["markdown"]?.stringValue
      == "![alt](https://images.example/a.png)\n[https://example.com](https://example.com)")
}

@Test(arguments: [PublicationHost.leaflet, .offprint, .pckt])
func nativeHostImages(_ host: PublicationHost) throws {
  let id = UUID()
  let image = PublishedBodyImage(
    assetID: id, blob: blob, alt: "image metadata", width: 0, height: -2,
    publicURL: "https://example.com/img")
  let result = try PublicationContentAdapter.prepare(
    document: doc("", blocks: [.image(assetID: id, alt: "block alt")]), host: host, images: [image])
  let object = try #require(result.content.objectValue)
  let block: [String: JSONValue]
  if host == .leaflet {
    block = try #require(
      object["pages"]?.arrayValue?.first?.objectValue?["blocks"]?.arrayValue?.first?.objectValue?[
        "block"]?.objectValue)
  } else {
    let first = try #require(object["items"]?.arrayValue?.first?.objectValue)
    block = host == .pckt ? try #require(first["attrs"]?.objectValue) : first
  }
  #expect(block["alt"] == .string("block alt"))
  #expect((block["image"] ?? block["blob"])?.blobCID == blob.ref.link)
  #expect(block["aspectRatio"] == .object(["width": .integer(1), "height": .integer(1)]))
  if host == .pckt { #expect(block["src"] == .string("blob:bafkexample")) }
  if host == .offprint {
    #expect(block["width"] == .string("100%"))
    #expect(block["alignment"] == .string("center"))
  }
}

@Test func missingAndDuplicateImagesThrowTypedErrors() {
  let id = UUID()
  #expect(throws: PublicationContentError(reason: "One or more body images are unavailable")) {
    try PublicationContentAdapter.prepare(
      document: doc("", blocks: [.image(assetID: id, alt: "missing")]), host: .leaflet)
  }
  let image = PublishedBodyImage(
    assetID: id, blob: blob, alt: "", width: 1, height: 1, publicURL: "https://example.com")
  #expect(throws: PublicationContentError(reason: "Body image IDs must be unique")) {
    try PublicationContentAdapter.prepare(
      document: doc(""), host: .leaflet, images: [image, image])
  }
}

@Test func leafletQuoteFacetOffsetsUseUTF8() throws {
  let lines = [
    RichText(plaintext: "👋", spans: [.init(byteStart: 0, byteEnd: 4, feature: .bold)]),
    RichText(
      plaintext: "é",
      spans: [.init(byteStart: 0, byteEnd: 2, feature: .link("https://example.com"))]),
  ]
  let result = try PublicationContentAdapter.prepare(
    document: doc("", blocks: [.quote(lines)]), host: .leaflet)
  let block = try #require(
    result.content.objectValue?["pages"]?.arrayValue?.first?.objectValue?["blocks"]?.arrayValue?
      .first?.objectValue?["block"]?.objectValue)
  #expect(block["plaintext"] == .string("👋\né"))
  let facets = try #require(block["facets"]?.arrayValue)
  #expect(
    facets[1].objectValue?["index"] == .object(["byteStart": .integer(5), "byteEnd": .integer(7)]))
  #expect(
    facets[1].objectValue?["features"]?.arrayValue?.first?.objectValue?["$type"]
      == .string("pub.leaflet.richtext.facet#link"))
}

@Test func pcktSummaryAndTerminalParagraph() throws {
  let result = try PublicationContentAdapter.prepare(
    document: doc("", blocks: [.thematicBreak]), host: .pckt, description: " Summary \n")
  let items = try #require(result.content.objectValue?["items"]?.arrayValue)
  #expect(items.count == 3)
  #expect(items.first?.objectValue?["level"] == .integer(3))
  #expect(items.last?.objectValue?["plaintext"] == .string(""))
  #expect(result.textContent == "Summary\nPlain")
  let noDuplicate = try PublicationContentAdapter.prepare(
    document: doc("", blocks: [.paragraph(.init(plaintext: "Summary"))]), host: .pckt,
    description: "Summary")
  #expect(noDuplicate.content.objectValue?["items"]?.arrayValue?.count == 1)
}

@Test(arguments: [PublicationHost.leaflet, .pckt])
func jsonOffloadWrappers(_ host: PublicationHost) throws {
  let result = try PublicationContentAdapter.prepare(
    document: doc(
      "", blocks: [.paragraph(.init(plaintext: String(repeating: "x", count: 110_000)))]),
    host: host)
  #expect(result.offload?.mimeType == "application/json")
  let replacement = try #require(result.replacingOffload(with: blob).objectValue)
  if host == .leaflet {
    #expect(replacement["pages"] == .array([]))
    #expect(replacement["blobs"] == .array([]))
    #expect(replacement["blobPages"]?.blobCID == blob.ref.link)
  } else {
    #expect(replacement["references"] == .array([]))
    #expect(replacement["blob"]?.blobCID == blob.ref.link)
    #expect(replacement["items"] == nil)
  }
}

@Test func offprintSizeLimitAndNativeHeading() throws {
  let heading = try PublicationContentAdapter.prepare(
    document: doc("", blocks: [.heading(level: 6, content: .init(plaintext: "Heading"))]),
    host: .offprint)
  #expect(
    heading.content.objectValue?["items"]?.arrayValue?.first?.objectValue?["level"] == .integer(3))
  #expect(throws: PublicationContentError.self) {
    try PublicationContentAdapter.prepare(
      document: doc(
        "", blocks: [.paragraph(.init(plaintext: String(repeating: "x", count: 900_000)))]),
      host: .offprint)
  }
}

@Test func nativeRecordWrappers() throws {
  let reference = StrongReference(uri: "at://did:plc:a/site.standard.document/a", cid: "cid")
  let offprint = OffprintArticleRecord(document: reference)
  let pckt = PcktDocumentRecord(document: reference, site: "publication")
  #expect(
    try JSONDecoder().decode(OffprintArticleRecord.self, from: JSONEncoder().encode(offprint))
      == offprint)
  #expect(
    try JSONDecoder().decode(PcktDocumentRecord.self, from: JSONEncoder().encode(pckt)) == pckt)
}

@Test(arguments: [PublicationHost.leaflet, .pckt])
func jsonOffloadExactByteThreshold(_ host: PublicationHost) throws {
  let threshold = host == .leaflet ? 102_400 : 20_000
  func prepared(_ length: Int) throws -> PreparedPublicationContent {
    try PublicationContentAdapter.prepare(
      document: doc(
        "", blocks: [.paragraph(.init(plaintext: String(repeating: "x", count: length)))]),
      host: host)
  }
  let baseline = try prepared(0)
  let key = host == .leaflet ? "pages" : "items"
  let overhead = try JSONEncoder().encode(try #require(baseline.content.objectValue?[key])).count
  let boundary = try prepared(threshold - overhead)
  #expect(boundary.offload == nil)
  #expect(try prepared(threshold - overhead + 1).offload != nil)
}

@Test func offprintExactByteThreshold() throws {
  func prepared(_ length: Int) throws -> PreparedPublicationContent {
    try PublicationContentAdapter.prepare(
      document: doc(
        "", blocks: [.paragraph(.init(plaintext: String(repeating: "x", count: length)))]),
      host: .offprint)
  }
  let overhead = try JSONEncoder().encode(prepared(0).content).count
  _ = try prepared(899_999 - overhead)
  #expect(throws: PublicationContentError.self) { try prepared(900_000 - overhead) }
}

@Test(arguments: [PublicationHost.leaflet, .offprint, .pckt])
func nativeListsAndCode(_ host: PublicationHost) throws {
  let list = CanonicalList(
    kind: .ordered, start: 4,
    items: [
      CanonicalListItem(
        content: .init(plaintext: "Parent"),
        children: [
          CanonicalList(kind: .unordered, items: [.init(content: .init(plaintext: "Child"))])
        ])
    ])
  let result = try PublicationContentAdapter.prepare(
    document: doc("", blocks: [.list(list), .code(language: " SWIFT \n", source: "let x = 1")]),
    host: host)
  let blocks: [JSONValue]
  if host == .leaflet {
    blocks = try #require(
      result.content.objectValue?["pages"]?.arrayValue?.first?.objectValue?["blocks"]?.arrayValue
    ).compactMap { $0.objectValue?["block"] }
  } else {
    blocks = try #require(result.content.objectValue?["items"]?.arrayValue)
  }
  let first = try #require(blocks.first?.objectValue)
  #expect(first[host == .leaflet ? "startIndex" : "start"] == .integer(4))
  let item = try #require(
    first[host == .pckt ? "content" : "children"]?.arrayValue?.first?.objectValue)
  if host == .leaflet { #expect(item["unorderedListChildren"] != nil) }
  if host == .offprint { #expect(item["children"]?.arrayValue?.count == 1) }
  if host == .pckt { #expect(item["content"]?.arrayValue?.count == 2) }
  #expect(blocks[1].objectValue?["language"] == .string("swift"))
  #expect(blocks[1].objectValue?[host == .offprint ? "code" : "plaintext"] == .string("let x = 1"))
}
