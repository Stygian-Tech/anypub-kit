import Foundation

public struct ATProtoBlobRef: Codable, Equatable, Sendable {
    public struct Link: Codable, Equatable, Sendable {
        public let link: String

        public init(link: String) { self.link = link }

        enum CodingKeys: String, CodingKey {
            case link = "$link"
        }
    }

    public let type: String
    public let ref: Link
    public let mimeType: String
    public let size: Int

    public init(type: String = "blob", ref: Link, mimeType: String, size: Int) {
        self.type = type; self.ref = ref; self.mimeType = mimeType; self.size = size
    }

    enum CodingKeys: String, CodingKey {
        case type = "$type"
        case ref
        case mimeType
        case size
    }
}

