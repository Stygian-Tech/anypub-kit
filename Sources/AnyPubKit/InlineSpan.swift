import Foundation

public struct InlineSpan: Equatable, Sendable {
    public let byteStart: Int
    public let byteEnd: Int
    public let feature: InlineFeature
    public init(byteStart: Int, byteEnd: Int, feature: InlineFeature) { self.byteStart = byteStart; self.byteEnd = byteEnd; self.feature = feature }
}

