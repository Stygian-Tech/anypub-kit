import AnyPubKit
import Foundation
import Testing

private let epoch = Date(timeIntervalSince1970: 1_700_000_000)
private let site = "at://did:plc:writer/site.standard.publication/blog"

@Test func explicitDatesAndCompatibilityInitializer() throws {
  let builder = StandardSiteRecordBuilder()
  let record = try builder.documentRecord(
    site: site, title: "Title", publishedAt: epoch, path: "/article", tags: [], langs: ["en"],
    content: .object(["$type": .string("at.markpub.markdown")]))
  #expect(record.publishedAt.date == epoch)
  #expect(record.updatedAt == nil)
  #expect(record.tags == [])
  #expect(record.langs == ["en"])
  let compatibility = StandardSiteDocumentRecord(
    site: site, title: "Title", publishedAt: ATProtoTimestamp(epoch), path: nil, tags: nil,
    langs: nil, coverImage: nil, description: nil, textContent: nil, content: nil, updatedAt: nil)
  let decoded = try JSONDecoder().decode(
    StandardSiteDocumentRecord.self, from: JSONEncoder().encode(compatibility))
  #expect(decoded == compatibility)
}

@Test func extensionsSurviveUpdatesAndRoundtrip() throws {
  let data = Data(
    #"{"$type":"site.standard.document","site":"https://example.com","title":"Original","publishedAt":"2024-01-01T00:00:00Z","labels":{"values":[{"val":"test"}]},"future":[true,null,42],"content":{"$type":"future.example.content","nested":{"unknown":true}}}"#
      .utf8)
  let original = try JSONDecoder().decode(StandardSiteDocumentRecord.self, from: data)
  let updated = try StandardSiteRecordBuilder().updatingDocument(original, updatedAt: epoch) {
    $0.title = "Updated"
    $0.publishedAt = ATProtoTimestamp(epoch)
  }
  #expect(updated.publishedAt == original.publishedAt)
  #expect(updated.updatedAt?.date == epoch)
  #expect(updated.additionalFields == original.additionalFields)
  #expect(updated.content == original.content)
  #expect(
    try JSONDecoder().decode(StandardSiteDocumentRecord.self, from: JSONEncoder().encode(updated))
      == updated)
}

@Test func publicationOptionalFieldsAndExtensions() throws {
  let builder = StandardSiteRecordBuilder()
  let record = try builder.publicationRecord(
    url: "https://example.com/blog", name: "Blog", description: "Description",
    preferences: .object(["showInDiscover": .bool(false), "future": .integer(9)]),
    additionalFields: ["theme": .object(["$type": .string("pub.leaflet.publication#theme")])])
  let updated = try builder.updatingPublication(record) { $0.name = "Renamed" }
  #expect(updated.additionalFields == record.additionalFields)
  #expect(updated.preferences == record.preferences)
  #expect(
    try JSONDecoder().decode(
      StandardSitePublicationRecord.self, from: JSONEncoder().encode(updated)) == updated)
  var collision = record
  collision.additionalFields["name"] = .string("Must not override")
  #expect(
    try JSONDecoder().decode(
      StandardSitePublicationRecord.self, from: JSONEncoder().encode(collision)
    ).name == "Blog")
}

@Test(arguments: [
  "http://example.com", "https://user:password@example.com", "https://example.com?q=x",
  "https://example.com/#fragment", "https://", "//example.com",
])
func rejectsPublicationURL(_ url: String) {
  #expect(throws: StandardSiteValidationError.self) {
    try StandardSiteRecordBuilder().publicationRecord(url: url, name: "Blog")
  }
}

@Test(arguments: [
  "../secret", "/a/../b", "/%2e%2e/b", "/%252e%252e/b", "//host/path", "https://host/path",
  "/a?x=1", "/a#fragment", "/a%3fb", "/a\\b", "/a b",
])
func rejectsUnsafePaths(_ path: String) {
  #expect(throws: StandardSiteValidationError.self) {
    try StandardSiteRecordBuilder().documentRecord(
      site: site, title: "Title", publishedAt: epoch, path: path)
  }
}

@Test(arguments: [
  "at://did:plc:a/site.standard.document/key", "at://did:plc:a/site.standard.publication/..",
  "at://did:plc:a/site.standard.publication/key/extra", "http://example.com",
])
func rejectsDocumentSites(_ value: String) {
  #expect(throws: StandardSiteValidationError.self) {
    try StandardSiteRecordBuilder().documentRecord(site: value, title: "Title", publishedAt: epoch)
  }
}

@Test func stringByteAndGraphemeBoundaries() throws {
  let builder = StandardSiteRecordBuilder()
  _ = try builder.documentRecord(
    site: site, title: String(repeating: "a", count: 500), publishedAt: epoch,
    tags: [String(repeating: "a", count: 128)], description: String(repeating: "d", count: 3_000))
  #expect(throws: StandardSiteValidationError.self) {
    try builder.documentRecord(
      site: site, title: String(repeating: "a", count: 501), publishedAt: epoch)
  }
  // Combining marks fit within one grapheme but exceed the byte limit.
  let hugeGrapheme = "a" + String(repeating: "\u{0301}", count: 2_500)
  #expect(hugeGrapheme.count == 1)
  #expect(throws: StandardSiteValidationError.self) {
    try builder.documentRecord(site: site, title: hugeGrapheme, publishedAt: epoch)
  }
  #expect(throws: StandardSiteValidationError.self) {
    try builder.documentRecord(
      site: site, title: "T", publishedAt: epoch, tags: [String(repeating: "x", count: 129)])
  }
  #expect(throws: StandardSiteValidationError.self) {
    try builder.documentRecord(
      site: site, title: "T", publishedAt: epoch, description: String(repeating: "d", count: 3_001))
  }
}

@Test(arguments: [
  JSONValue.null, .string("markdown"), .object([:]), .object(["$type": .string("invalid")]),
  .object(["$type": .string("a.b.c#")]),
  .object(["$type": .string("at.markpub.markdown#main")]),
])
func rejectsInvalidContent(_ content: JSONValue) {
  #expect(throws: StandardSiteValidationError.self) {
    try StandardSiteRecordBuilder().documentRecord(
      site: site, title: "Title", publishedAt: epoch, content: content)
  }
}

@Test func timestampsAndJSONNumbers() throws {
  for value in ["2024-01-01T00:00:00Z", "2024-01-01T00:00:00.000Z"] {
    let stamp = try JSONDecoder().decode(ATProtoTimestamp.self, from: JSONEncoder().encode(value))
    #expect(
      try JSONDecoder().decode(ATProtoTimestamp.self, from: JSONEncoder().encode(stamp)) == stamp)
  }
  #expect(throws: DecodingError.self) {
    try JSONDecoder().decode(ATProtoTimestamp.self, from: Data("\"no\"".utf8))
  }
  #expect(JSONValue.number(.infinity).integerValue == nil)
}

@Test func didWebAuthorityAndEncodedRelativePath() throws {
  let builder = StandardSiteRecordBuilder()
  let record = try builder.documentRecord(
    site: "at://did:web:example.com%3A8443/site.standard.publication/key", title: "Title",
    publishedAt: epoch, path: "/caf%C3%A9")
  #expect(record.path == "/caf%C3%A9")
  #expect(throws: StandardSiteValidationError.self) {
    try builder.documentRecord(
      site: "at://did:web:example.com%zz/site.standard.publication/key", title: "Title",
      publishedAt: epoch)
  }
}

@Test func descriptionAndTagByteLimits() throws {
  let builder = StandardSiteRecordBuilder()
  let tag = "a" + String(repeating: "\u{0301}", count: 640)
  let description = "a" + String(repeating: "\u{0301}", count: 15_000)
  #expect(throws: StandardSiteValidationError.self) {
    try builder.documentRecord(site: site, title: "T", publishedAt: epoch, tags: [tag])
  }
  #expect(throws: StandardSiteValidationError.self) {
    try builder.publicationRecord(
      url: "https://example.com", name: "Blog", description: description)
  }
  #expect(throws: StandardSiteValidationError.self) {
    try builder.publicationRecord(
      url: "https://example.com", name: String(repeating: "n", count: 501))
  }
}
