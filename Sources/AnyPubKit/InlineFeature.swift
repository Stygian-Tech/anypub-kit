import Foundation

public enum InlineFeature: Equatable, Sendable {
    case bold
    case italic
    case code
    case strikethrough
    case underline
    case link(String)
}

