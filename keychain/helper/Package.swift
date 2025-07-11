// swift-tools-version: 6.1
// The swift-tools-version declares the minimum version of Swift required to build this package.

import PackageDescription

let package = Package(
    name: "keychainHelper",
    products: [
        .executable(name: "keychainHelperClient", targets: ["Client"]),
        .executable(name: "keychainHelperService", targets: ["XPCService"]),
        //.executable(name: "KeychainMachPortHelper", targets: ["MachPort"]),
    ],
    targets: [
        .target(name: "KeychainHelperProtocol"),
        .target(
            name: "Service",
            dependencies: ["KeychainHelperProtocol"]),
        .executableTarget(
            name: "Client",
            dependencies: ["KeychainHelperProtocol"]),
        .executableTarget(
            name: "XPCService",
            dependencies: ["Service", "KeychainHelperProtocol"]),
        //.executableTarget(
        //    name: "MachPort",
        //    dependencies: ["Service", "KeychainHelperProtocol"]),
    ]
)
