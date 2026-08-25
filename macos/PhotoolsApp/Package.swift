// swift-tools-version: 5.9

import PackageDescription

let package = Package(
    name: "PhotoolsApp",
    platforms: [
        .macOS(.v13)
    ],
    products: [
        .executable(name: "PhotoolsApp", targets: ["PhotoolsApp"]),
        .library(name: "PhotoolsCore", targets: ["PhotoolsCore"])
    ],
    targets: [
        .target(
            name: "PhotoolsCore"
        ),
        .executableTarget(
            name: "PhotoolsApp",
            dependencies: ["PhotoolsCore"]
        ),
        .testTarget(
            name: "PhotoolsCoreTests",
            dependencies: ["PhotoolsCore"]
        )
    ]
)
