import Foundation

public struct CanonicalList: Equatable, Sendable {
    public let kind: CanonicalListKind
    public let start: Int?
    public var items: [CanonicalListItem]
    public init(kind: CanonicalListKind, start: Int? = nil, items: [CanonicalListItem]) { self.kind = kind; self.start = start; self.items = items }
}

