// KeychainClient/main.swift

import Foundation
import KeychainHelperProtocol

print("KeychainClient started")

// --- Command-Line Argument Parsing ---
let arguments = CommandLine.arguments

// Check for minimum required arguments
guard arguments.count > 1 else {
    printUsage()
    exit(1)
}

// Default to machServiceName unless --service-name is specified
var useServiceName = false
var argIndex = 1

// Check for connection type flag
if arguments[1] == "--service-name" {
    useServiceName = true
    argIndex = 2

    // Make sure we still have enough arguments after the flag
    guard arguments.count > argIndex else {
        printUsage()
        exit(1)
    }
}

// --- XPC Connection Setup ---
let connection: NSXPCConnection
if useServiceName {
    print("Using serviceName connection")
    connection = NSXPCConnection(serviceName: "io.cloudeng.KeychainHelper")
} else {
    print("Using machServiceName connection")
    connection = NSXPCConnection(machServiceName: "io.cloudeng.KeychainHelper")
}

connection.remoteObjectInterface = NSXPCInterface(with: KeychainXPCProtocol.self)
connection.resume()

let service =
    connection.remoteObjectProxyWithErrorHandler { error in
        print("Error connecting to XPC service: \(error.localizedDescription)")
        exit(1)
    } as! KeychainXPCProtocol

// Get the command after possibly skipping the connection type flag
let command = arguments[argIndex]
argIndex += 1

switch command {

case "get":
    guard arguments.count >= argIndex + 3 else {
        print("Error: 'get' command requires account, service and label arguments.")
        printUsage()
        exit(1)
    }
    let accountName = arguments[argIndex]
    let serviceName = arguments[argIndex + 1]
    let label: String = arguments[argIndex + 2]

    service.get(account: accountName, service: serviceName, label: label) { value, error in
        if let error = error {
            print("Failed to read from keychain: \(error)")
            exit(1)
        }
        if let value = value {
            print(value)
        } else {
            print("No value found for the specified service and account.")
        }
        exit(0)
    }

default:
    print("Error: Unknown command '\(command)'.")
    printUsage()
    exit(1)
}

func printUsage() {
    print(
        """
        Usage:
          KeychainClient [--service-name] <command> <args>

        Connection options:
          --service-name    Use serviceName connection instead of machServiceName (default)

        Commands:
          get <service> <account> <label> Retrieve data from keychain
        """)
}

// Keep the application running to allow the async XPC call to complete.
dispatchMain()
