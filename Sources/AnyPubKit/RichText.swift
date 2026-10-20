import Foundation

public struct RichText: Equatable, Sendable {
    public let plaintext: String
    public let spans: [InlineSpan]
    public init(plaintext: String, spans: [InlineSpan] = []) { self.plaintext = plaintext; self.spans = spans }
}

