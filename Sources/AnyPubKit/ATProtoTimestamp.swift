import Foundation

public struct ATProtoTimestamp: Codable, Equatable, Sendable {
    public let date: Date

    public init(_ date: Date) {
        self.date = date
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.singleValueContainer()
        let value = try container.decode(String.self)
        let fractional = ISO8601DateFormatter()
        fractional.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        let standard = ISO8601DateFormatter()
        standard.formatOptions = [.withInternetDateTime]
        guard let date = fractional.date(from: value) ?? standard.date(from: value) else {
            throw DecodingError.dataCorruptedError(
                in: container,
                debugDescription: "Invalid AT Protocol timestamp"
            )
        }
        self.date = date
    }

    public func encode(to encoder: Encoder) throws {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        formatter.timeZone = TimeZone(secondsFromGMT: 0)
        var container = encoder.singleValueContainer()
        try container.encode(formatter.string(from: date))
    }
}

