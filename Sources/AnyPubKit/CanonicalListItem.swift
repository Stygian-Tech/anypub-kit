import Foundation

public struct CanonicalListItem: Equatable, Sendable {
    public let content: RichText
    public let checked: Bool?
    public var children: [CanonicalList]
    public init(content: RichText, checked: Bool? = nil, children: [CanonicalList] = []) { self.content = content; self.checked = checked; self.children = children }
}

