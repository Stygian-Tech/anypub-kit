import Foundation

public struct StrongReference: Codable, Equatable, Sendable {
    public let type: String?
    public let uri: String
    public let cid: String

    public init(uri: String, cid: String, type: String? = nil) {
        self.type = type
        self.uri = uri
        self.cid = cid
    }

    enum CodingKeys: String, CodingKey {
        case type = "$type"
        case uri
        case cid
    }
}

public struct OffprintArticleRecord: Codable, Equatable, Sendable {
    public let type = "app.offprint.document.article"
    public let document: StrongReference

    public init(document: StrongReference) { self.document = document }

    enum CodingKeys: String, CodingKey {
        case type = "$type"
        case document
    }
}

public struct PcktDocumentRecord: Codable, Equatable, Sendable {
    public let type = "blog.pckt.document"
    public let document: StrongReference
    public let site: String

    public init(document: StrongReference, site: String) { self.document = document; self.site = site }

    enum CodingKeys: String, CodingKey {
        case type = "$type"
        case document
        case site
    }
}

