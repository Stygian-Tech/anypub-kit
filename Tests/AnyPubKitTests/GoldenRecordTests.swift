import AnyPubKit
import Foundation
import Testing

private struct GoldenRecord: Decodable {
  let name: String
  let kind: String
  let input: Input
  let expected: JSONValue

  struct Input: Decodable {
    let url: String?
    let name: String?
    let site: String?
    let title: String?
    let publishedAt: ATProtoTimestamp?
    let updatedAt: ATProtoTimestamp?
    let path: String?
    let tags: [String]?
    let description: String?
    let textContent: String?
    let content: JSONValue?
    let preferences: JSONValue?
  }
}

@Test func buildersMatchSharedGoldenRecords() throws {
  // Read the same repository fixture as TypeScript; do not copy it into test resources.
  let root = URL(fileURLWithPath: #filePath)
    .deletingLastPathComponent().deletingLastPathComponent().deletingLastPathComponent()
  let fixtures = try JSONDecoder().decode(
    [GoldenRecord].self,
    from: Data(contentsOf: root.appendingPathComponent("Fixtures/standard-site.json")))
  #expect(Set(fixtures.map(\.kind)) == ["publication", "document"])
  let builder = StandardSiteRecordBuilder()
  for fixture in fixtures {
    let input = fixture.input
    let encoded: Data
    switch fixture.kind {
    case "publication":
      encoded = try JSONEncoder().encode(
        builder.publicationRecord(
          url: #require(input.url), name: #require(input.name), description: input.description,
          preferences: input.preferences))
    case "document":
      encoded = try JSONEncoder().encode(
        builder.documentRecord(
          site: #require(input.site), title: #require(input.title),
          publishedAt: #require(input.publishedAt).date, path: input.path, tags: input.tags,
          description: input.description, textContent: input.textContent, content: input.content,
          updatedAt: input.updatedAt?.date))
    default:
      Issue.record("Unknown fixture kind: \(fixture.kind)")
      continue
    }
    #expect(
      try JSONDecoder().decode(JSONValue.self, from: encoded) == fixture.expected,
      "Builder output differs for \(fixture.name)")
  }
}

@Test(arguments: [
  ("https://example.com/", "https://example.com"),
  ("https://example.com/blog///", "https://example.com/blog"),
  ("https://example.com/blog", "https://example.com/blog"),
  ("https://example.com/%2F/", "https://example.com/%2F"),
])
func publicationBuildersNormalizeTrailingSlashes(input: String, expected: String) throws {
  let builder = StandardSiteRecordBuilder()
  let built = try builder.publicationRecord(url: input, name: "Blog")
  #expect(built.url == expected)
  let updated = try builder.updatingPublication(built) { $0.url = input }
  #expect(updated.url == expected)
  let decoded = StandardSitePublicationRecord(url: input, name: "Blog")
  #expect(try builder.updatingPublication(decoded) { $0.name = "Renamed" }.url == expected)
}

@Test(arguments: [
  "https:///", "https:////", "https://?q=/", "https://example.com/#/", "https://user@example.com/",
])
func publicationBuildersValidateOriginalURL(input: String) throws {
  let builder = StandardSiteRecordBuilder()
  let original = try builder.publicationRecord(url: "https://example.com", name: "Blog")
  let error = StandardSiteValidationError.invalidField(
    field: "url", reason: "Expected an HTTPS publication URL")
  #expect(throws: error) { try builder.publicationRecord(url: input, name: "Blog") }
  #expect(throws: error) { try builder.updatingPublication(original) { $0.url = input } }
}

@Test func discoveryWrappersPreserveWirePayloads() throws {
  let reference = StrongReference(
    uri: "at://did:plc:writer/site.standard.document/post", cid: "bafyexample")
  let offprint = OffprintArticleRecord(document: reference)
  let pckt = PcktDocumentRecord(
    document: reference, site: "at://did:plc:writer/blog.pckt.publication/blog")
  let document: JSONValue = .object(["uri": .string(reference.uri), "cid": .string(reference.cid)])
  #expect(
    try JSONDecoder().decode(JSONValue.self, from: JSONEncoder().encode(offprint))
      == .object([
        "$type": .string("app.offprint.document.article"), "document": document,
      ]))
  #expect(
    try JSONDecoder().decode(JSONValue.self, from: JSONEncoder().encode(pckt))
      == .object([
        "$type": .string("blog.pckt.document"), "document": document, "site": .string(pckt.site),
      ]))
  #expect(
    try JSONDecoder().decode(OffprintArticleRecord.self, from: JSONEncoder().encode(offprint))
      == offprint)
  #expect(
    try JSONDecoder().decode(PcktDocumentRecord.self, from: JSONEncoder().encode(pckt)) == pckt)
}
