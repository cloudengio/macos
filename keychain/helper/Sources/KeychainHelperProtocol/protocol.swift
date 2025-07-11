import Foundation

@objc public protocol KeychainXPCProtocol {
    func get(
        account: String, service: String, label: String,
        withReply reply: @escaping (String?, String?) -> Void)
}
