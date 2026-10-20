// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "AnyPubKit",
    platforms: [.macOS(.v14), .iOS(.v17)],
    products: [.library(name: "AnyPubKit", targets: ["AnyPubKit"])],
    targets: [
        .target(name: "AnyPubKit"),
        .testTarget(name: "AnyPubKitTests", dependencies: ["AnyPubKit"]),
    ]
)
