import Foundation

/// A wire record. Use StandardSiteRecordBuilder for validated construction.
public struct StandardSiteDocumentRecord: Codable, Equatable, Sendable {
    public let type = "site.standard.document"
    public var site: String
    public var title: String
    public var publishedAt: ATProtoTimestamp
    public var path: String?
    public var tags: [String]?
    public var langs: [String]?
    public var coverImage: ATProtoBlobRef?
    public var description: String?
    public var textContent: String?
    public var content: JSONValue?
    public var updatedAt: ATProtoTimestamp?
    /// Unrecognized top-level fields, retained when decoding and updating records.
    public var additionalFields: [String: JSONValue]

    public init(
        site: String,
        title: String,
        publishedAt: ATProtoTimestamp,
        path: String? = nil,
        tags: [String]? = nil,
        langs: [String]? = nil,
        coverImage: ATProtoBlobRef? = nil,
        description: String? = nil,
        textContent: String? = nil,
        content: JSONValue? = nil,
        updatedAt: ATProtoTimestamp? = nil,
        additionalFields: [String: JSONValue] = [:]
    ) {
        self.site = site
        self.title = title
        self.publishedAt = publishedAt
        self.path = path
        self.tags = tags
        self.langs = langs
        self.coverImage = coverImage
        self.description = description
        self.textContent = textContent
        self.content = content
        self.updatedAt = updatedAt
        self.additionalFields = additionalFields
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: RecordKey.self)
        guard try container.decode(String.self, forKey: RecordKey("$type")) == "site.standard.document" else {
            throw DecodingError.dataCorruptedError(forKey: RecordKey("$type"), in: container, debugDescription: "Unexpected record type")
        }
        site = try container.decode(String.self, forKey: RecordKey("site"))
        title = try container.decode(String.self, forKey: RecordKey("title"))
        publishedAt = try container.decode(ATProtoTimestamp.self, forKey: RecordKey("publishedAt"))
        path = try container.decodeIfPresent(String.self, forKey: RecordKey("path"))
        tags = try container.decodeIfPresent([String].self, forKey: RecordKey("tags"))
        langs = try container.decodeIfPresent([String].self, forKey: RecordKey("langs"))
        coverImage = try container.decodeIfPresent(ATProtoBlobRef.self, forKey: RecordKey("coverImage"))
        description = try container.decodeIfPresent(String.self, forKey: RecordKey("description"))
        textContent = try container.decodeIfPresent(String.self, forKey: RecordKey("textContent"))
        content = try container.decodeIfPresent(JSONValue.self, forKey: RecordKey("content"))
        updatedAt = try container.decodeIfPresent(ATProtoTimestamp.self, forKey: RecordKey("updatedAt"))
        let known: Set<String> = ["$type", "site", "title", "publishedAt", "path", "tags", "langs", "coverImage", "description", "textContent", "content", "updatedAt"]
        additionalFields = [:]
        for key in container.allKeys where !known.contains(key.stringValue) {
            additionalFields[key.stringValue] = try container.decode(JSONValue.self, forKey: key)
        }
    }

    public func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: RecordKey.self)
        let known: Set<String> = ["$type", "site", "title", "publishedAt", "path", "tags", "langs", "coverImage", "description", "textContent", "content", "updatedAt"]
        for (key, value) in additionalFields where !known.contains(key) {
            try container.encode(value, forKey: RecordKey(key))
        }
        try container.encode(type, forKey: RecordKey("$type"))
        try container.encode(site, forKey: RecordKey("site"))
        try container.encode(title, forKey: RecordKey("title"))
        try container.encode(publishedAt, forKey: RecordKey("publishedAt"))
        try container.encodeIfPresent(path, forKey: RecordKey("path"))
        try container.encodeIfPresent(tags, forKey: RecordKey("tags"))
        try container.encodeIfPresent(langs, forKey: RecordKey("langs"))
        try container.encodeIfPresent(coverImage, forKey: RecordKey("coverImage"))
        try container.encodeIfPresent(description, forKey: RecordKey("description"))
        try container.encodeIfPresent(textContent, forKey: RecordKey("textContent"))
        try container.encodeIfPresent(content, forKey: RecordKey("content"))
        try container.encodeIfPresent(updatedAt, forKey: RecordKey("updatedAt"))
    }
}
