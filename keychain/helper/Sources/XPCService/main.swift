import Foundation
import Service
import os

let log = OSLog(subsystem: "io.cloudeng.KeychainHelper", category: "debug")
os_log("XPC Service started", log: log, type: .info)

func isRunningFromApplicationsAppBundle() -> Bool {
    // Get the executable path
    let executablePath = CommandLine.arguments[0]
    let url = URL(fileURLWithPath: executablePath).standardized

    // Traverse up the directory tree to find the .app bundle
    var currentURL = url
    while currentURL.path != "/" {
        if currentURL.pathExtension == "app" {
            // Found the .app bundle, check if it is under /Applications
            let appBundlePath = currentURL.path
            if appBundlePath.hasPrefix("/Applications/") || appBundlePath == "/Applications" {
                return true
            }
        }
        currentURL.deleteLastPathComponent()
    }
    return false
}

let delegate = ServiceDelegate()

var listener: NSXPCListener

func waitForDebugger() {
    os_log("Waiting for debugger...", log: log, type: .info)
    while !isDebuggerAttached() {
        usleep(100_000)  // 100 ms
    }
    os_log("Debugger attached!", log: log, type: .info)
}

func isDebuggerAttached() -> Bool {
    var info = kinfo_proc()
    var size = MemoryLayout<kinfo_proc>.stride
    var name: [Int32] = [CTL_KERN, KERN_PROC, KERN_PROC_PID, getpid()]
    let result = sysctl(&name, u_int(name.count), &info, &size, nil, 0)
    return result == 0 && (info.kp_proc.p_flag & P_TRACED) != 0
}

//waitForDebugger()

if isRunningFromApplicationsAppBundle() {
    os_log("Running from Applications app bundle, using service listener", log: log, type: .info)
    listener = NSXPCListener.service()
} else {
    os_log(
        "Running from non-Applications app bundle, using machServiceName listener", log: log,
        type: .info)
    listener = NSXPCListener(machServiceName: "io.cloudeng.KeychainHelper")
}

listener.delegate = delegate
listener.resume()
