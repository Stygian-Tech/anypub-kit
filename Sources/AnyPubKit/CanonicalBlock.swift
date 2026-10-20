import Foundation

public enum CanonicalBlock: Equatable, Sendable {
  case paragraph(RichText)
  case heading(level: Int, content: RichText)
  case quote([RichText])
  case list(CanonicalList)
  case code(language: String?, source: String)
  case image(assetID: UUID, alt: String)
  case embed(url: String)
  case thematicBreak
}
