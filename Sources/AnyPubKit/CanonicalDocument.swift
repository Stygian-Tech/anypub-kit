import Foundation

public struct CanonicalDocument: Equatable, Sendable {
    public let blocks: [CanonicalBlock]
    public let plaintext: String
    public let markdown: String
    public init(blocks: [CanonicalBlock], plaintext: String, markdown: String) { self.blocks = blocks; self.plaintext = plaintext; self.markdown = markdown }
}

