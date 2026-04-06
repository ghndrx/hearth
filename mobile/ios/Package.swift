// swift-tools-version: 5.9

import PackageDescription

let package = Package(
    name: "Hearth",
    platforms: [
        .iOS(.v16),
    ],
    products: [
        .library(
            name: "Hearth",
            targets: ["Hearth"]
        ),
    ],
    targets: [
        .target(
            name: "Hearth",
            path: "Hearth"
        ),
        .testTarget(
            name: "HearthTests",
            dependencies: ["Hearth"],
            path: "HearthTests"
        ),
    ]
)
