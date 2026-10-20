public struct PublicationContentError: Error, Equatable, Sendable {
    public let reason: String
    public init(reason: String) { self.reason = reason }
}
