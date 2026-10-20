import Foundation

/// A wire record. Use StandardSiteRecordBuilder for validated construction.
public struct StandardSitePublicationRecord: Codable, Equatable, Sendable {
    public let type = "site.standard.publication"
    public var url: String
    public var name: String
    public var icon: ATProtoBlobRef?
    public var description: String?
    public var basicTheme: JSONValue?
    public var labels: JSONValue?
    public var preferences: JSONValue?
    /// Unrecognized top-level fields, retained when decoding and updating records.
    public var additionalFields: [String: JSONValue]

    public init(
        url: String,
        name: String,
        icon: ATProtoBlobRef? = nil,
        description: String? = nil,
        basicTheme: JSONValue? = nil,
        labels: JSONValue? = nil,
        preferences: JSONValue? = nil,
        additionalFields: [String: JSONValue] = [:]
    ) {
        self.url = url
        self.name = name
        self.icon = icon
        self.description = description
        self.basicTheme = basicTheme
        self.labels = labels
        self.preferences = preferences
        self.additionalFields = additionalFields
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: RecordKey.self)
        guard try container.decode(String.self, forKey: RecordKey("$type")) == "site.standard.publication" else {
            throw DecodingError.dataCorruptedError(forKey: RecordKey("$type"), in: container, debugDescription: "Unexpected record type")
        }
        url = try container.decode(String.self, forKey: RecordKey("url"))
        name = try container.decode(String.self, forKey: RecordKey("name"))
        icon = try container.decodeIfPresent(ATProtoBlobRef.self, forKey: RecordKey("icon"))
        description = try container.decodeIfPresent(String.self, forKey: RecordKey("description"))
        basicTheme = try container.decodeIfPresent(JSONValue.self, forKey: RecordKey("basicTheme"))
        labels = try container.decodeIfPresent(JSONValue.self, forKey: RecordKey("labels"))
        preferences = try container.decodeIfPresent(JSONValue.self, forKey: RecordKey("preferences"))
        let known: Set<String> = ["$type", "url", "name", "icon", "description", "basicTheme", "labels", "preferences"]
        additionalFields = [:]
        for key in container.allKeys where !known.contains(key.stringValue) {
            additionalFields[key.stringValue] = try container.decode(JSONValue.self, forKey: key)
        }
    }

    public func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: RecordKey.self)
        let known: Set<String> = ["$type", "url", "name", "icon", "description", "basicTheme", "labels", "preferences"]
        for (key, value) in additionalFields where !known.contains(key) {
            try container.encode(value, forKey: RecordKey(key))
        }
        try container.encode(type, forKey: RecordKey("$type"))
        try container.encode(url, forKey: RecordKey("url"))
        try container.encode(name, forKey: RecordKey("name"))
        try container.encodeIfPresent(icon, forKey: RecordKey("icon"))
        try container.encodeIfPresent(description, forKey: RecordKey("description"))
        try container.encodeIfPresent(basicTheme, forKey: RecordKey("basicTheme"))
        try container.encodeIfPresent(labels, forKey: RecordKey("labels"))
        try container.encodeIfPresent(preferences, forKey: RecordKey("preferences"))
    }
}
