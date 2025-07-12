import Foundation
import KeychainHelperProtocol
import os

let log = OSLog(subsystem: "io.cloudeng.KeychainHelper", category: "debug")

func keychainErrorMessage(status: OSStatus) -> String {
    if let cfStr = SecCopyErrorMessageString(status, nil) {
        return cfStr as String
    }
    return "Keychain error: \(status)"
}

// Implement it
class KeychainXPC: NSObject, KeychainXPCProtocol {
    func get(
        account: String, service: String, label: String,
        withReply reply: @escaping (String?, String?) -> Void
    ) {
        os_log(
            "KeychainXPC: Attempting to get value for account: %{public}@, service: %{public}@",
            log: log,
            type: .info, account, service)

        let query: [CFString: Any] = [
            kSecClass: kSecClassGenericPassword,
            kSecAttrAccessGroup: "84RRJLK9H4.io.cloudeng.KeychainHelper",
            kSecAttrAccount: account,
            kSecAttrService: service,
            kSecAttrLabel: label,
            kSecAttrSynchronizable: kSecAttrSynchronizableAny,
            kSecReturnData: true,
            kSecMatchLimit: kSecMatchLimitOne,
        ]
        os_log(
            "KeychainXPC: Querying keychain with query: %{public}@", log: log, type: .info,
            query as NSDictionary)
        var item: CFTypeRef?
        //usleep(20_000_000)
        let status = SecItemCopyMatching(query as CFDictionary, &item)
        os_log(
            "KeychainXPC: SecItemCopyMatching returned status: %{public}s", log: log, type: .info,
            keychainErrorMessage(status: status))
        if status == errSecSuccess,
            let data = item as? Data,
            let value = String(data: data, encoding: .utf8)
        {
            os_log(
                "KeychainXPC: Successfully retrieved value for account: %{public}@, service: %{public}@",
                log: log,
                type: .info,
                account, service
            )
            reply(value, nil)
        } else {
            os_log(
                "KeychainXPC: Failed to get value. Status: %{public}s", log: log, type: .error,
                keychainErrorMessage(status: status))
            reply(nil, keychainErrorMessage(status: status))
        }
    }
}

public class ServiceDelegate: NSObject, NSXPCListenerDelegate {
    public func listener(
        _ listener: NSXPCListener, shouldAcceptNewConnection connection: NSXPCConnection
    )
        -> Bool
    {
        os_log("KeychainXPC: Received new connection request", log: log, type: .info)
        connection.exportedInterface = NSXPCInterface(with: KeychainXPCProtocol.self)
        connection.exportedObject = KeychainXPC()
        connection.resume()
        os_log("KeychainXPC: Connection accepted and resumed", log: log, type: .info)
        return true
    }
}
