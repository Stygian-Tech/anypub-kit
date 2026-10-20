import Foundation

public enum StandardSiteValidationError: Error, Equatable, Sendable {
  case invalidField(field: String, reason: String)
}

/// Stateless construction: callers own lifecycle dates, persistence, and blob uploads.
public struct StandardSiteRecordBuilder: Sendable {
  public init() {}

  public func documentRecord(
    site: String, title: String, publishedAt: Date,
    path: String? = nil, tags: [String]? = nil, langs: [String]? = nil,
    coverImage: ATProtoBlobRef? = nil, description: String? = nil,
    textContent: String? = nil, content: JSONValue? = nil, updatedAt: Date? = nil,
    additionalFields: [String: JSONValue] = [:]
  ) throws -> StandardSiteDocumentRecord {
    let record = StandardSiteDocumentRecord(
      site: site, title: title, publishedAt: ATProtoTimestamp(publishedAt),
      path: path, tags: tags, langs: langs, coverImage: coverImage,
      description: description, textContent: textContent, content: content,
      updatedAt: updatedAt.map(ATProtoTimestamp.init), additionalFields: additionalFields
    )
    try validate(record)
    return record
  }

  public func publicationRecord(
    url: String, name: String, icon: ATProtoBlobRef? = nil,
    description: String? = nil, basicTheme: JSONValue? = nil,
    labels: JSONValue? = nil, preferences: JSONValue? = nil,
    additionalFields: [String: JSONValue] = [:]
  ) throws -> StandardSitePublicationRecord {
    let record = StandardSitePublicationRecord(
      url: url, name: name, icon: icon, description: description,
      basicTheme: basicTheme, labels: labels, preferences: preferences,
      additionalFields: additionalFields
    )
    try validate(record)
    return record
  }

  /// Applies edits to a decoded record while retaining extensions and its original publication date.
  public func updatingDocument(
    _ original: StandardSiteDocumentRecord, updatedAt: Date,
    changes: (inout StandardSiteDocumentRecord) throws -> Void
  ) throws -> StandardSiteDocumentRecord {
    var record = original
    try changes(&record)
    record.publishedAt = original.publishedAt
    record.updatedAt = ATProtoTimestamp(updatedAt)
    try validate(record)
    return record
  }

  public func updatingPublication(
    _ original: StandardSitePublicationRecord,
    changes: (inout StandardSitePublicationRecord) throws -> Void
  ) throws -> StandardSitePublicationRecord {
    var record = original
    try changes(&record)
    try validate(record)
    return record
  }

  public func validate(_ record: StandardSiteDocumentRecord) throws {
    try text(record.title, field: "title", bytes: 5_000, graphemes: 500)
    if let description = record.description {
      try text(description, field: "description", bytes: 30_000, graphemes: 3_000)
    }
    for tag in record.tags ?? [] { try text(tag, field: "tags", bytes: 1_280, graphemes: 128) }
    guard validHTTPS(record.site) || validPublicationURI(record.site) else {
      throw invalid("site", "Expected an AT publication URI or HTTPS URL")
    }
    if let path = record.path, !validPath(path) {
      throw invalid(
        "path", "Expected a relative path without traversal, query, fragment, or authority")
    }
    if let content = record.content {
      guard let type = content.objectValue?["$type"]?.stringValue, validType(type) else {
        throw invalid("content", "Expected an object with a valid Lexicon $type")
      }
    }
  }

  public func validate(_ record: StandardSitePublicationRecord) throws {
    guard validHTTPS(record.url) else { throw invalid("url", "Expected an HTTPS publication URL") }
    try text(record.name, field: "name", bytes: 5_000, graphemes: 500)
    if let description = record.description {
      try text(description, field: "description", bytes: 30_000, graphemes: 3_000)
    }
  }

  private func invalid(_ field: String, _ reason: String) -> StandardSiteValidationError {
    .invalidField(field: field, reason: reason)
  }

  private func text(_ value: String, field: String, bytes: Int, graphemes: Int) throws {
    guard value.utf8.count <= bytes, value.count <= graphemes else {
      throw invalid(field, "Exceeds \(bytes) UTF-8 bytes or \(graphemes) graphemes")
    }
  }

  private func validHTTPS(_ value: String) -> Bool {
    guard clean(value), let url = URLComponents(string: value),
      url.scheme?.lowercased() == "https", let host = url.host, !host.isEmpty,
      url.user == nil, url.password == nil, url.query == nil, url.fragment == nil
    else { return false }
    return true
  }

  private func validPublicationURI(_ value: String) -> Bool {
    guard clean(value), value.hasPrefix("at://"),
      !value.contains("?"), !value.contains("#"), !value.contains("@")
    else { return false }
    let parts = value.dropFirst(5).split(separator: "/", omittingEmptySubsequences: false)
    guard parts.count == 3, !parts[0].isEmpty, parts[1] == "site.standard.publication" else {
      return false
    }
    let authority = String(parts[0])
    let validAuthority =
      authority.hasPrefix("did:")
      ? authority.range(
        of: #"^did:[a-z]+:(?:[A-Za-z0-9._:-]|%[0-9A-Fa-f]{2})+$"#, options: .regularExpression)
        != nil
      : authority.range(
        of: #"^[A-Za-z0-9]([A-Za-z0-9.-]*[A-Za-z0-9])?\.[A-Za-z]{2,}$"#, options: .regularExpression
      ) != nil
    return validAuthority && parts[2] != "." && parts[2] != ".." && parts[2].count <= 512
      && parts[2].range(of: #"^[A-Za-z0-9._~:-]+$"#, options: .regularExpression) != nil
  }

  private func validPath(_ value: String) -> Bool {
    guard !value.isEmpty, clean(value), !value.hasPrefix("//"),
      let decoded = value.removingPercentEncoding, clean(decoded),
      !decoded.hasPrefix("//"), !decoded.contains("\\"), !decoded.contains("?"),
      !decoded.contains("#"), !decoded.contains(":"), !decoded.contains("%")
    else { return false }
    return !decoded.split(separator: "/", omittingEmptySubsequences: false).contains(where: {
      $0 == "." || $0 == ".."
    })
  }

  private func clean(_ value: String) -> Bool {
    !value.unicodeScalars.contains {
      CharacterSet.whitespacesAndNewlines.contains($0)
        || CharacterSet.controlCharacters.contains($0)
    }
  }

  private func validType(_ value: String) -> Bool {
    let pieces = value.split(separator: "#", omittingEmptySubsequences: false)
    guard pieces.count <= 2 else { return false }
    let parts = pieces[0].split(separator: ".", omittingEmptySubsequences: false)
    guard parts.count >= 3, pieces[0].utf8.count <= 317 else { return false }
    for (index, part) in parts.enumerated() {
      let pattern =
        index == parts.count - 1
        ? #"^[A-Za-z][A-Za-z0-9]*$"# : #"^[A-Za-z]([A-Za-z0-9-]*[A-Za-z0-9])?$"#
      guard part.utf8.count <= 63, part.range(of: pattern, options: .regularExpression) != nil
      else { return false }
    }
    return pieces.count == 1
      || pieces[1].range(of: #"^[A-Za-z][A-Za-z0-9_]*$"#, options: .regularExpression) != nil
  }
}
