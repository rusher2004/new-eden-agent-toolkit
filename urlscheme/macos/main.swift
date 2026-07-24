import Cocoa
import Carbon

func logFailure(_ message: String) {
    let logPath = NSHomeDirectory() + "/.neat/run/callback-handler.log"
    try? FileManager.default.createDirectory(
        atPath: NSHomeDirectory() + "/.neat/run", withIntermediateDirectories: true)
    let timestamp = ISO8601DateFormatter().string(from: Date())
    let line = "[\(timestamp)] \(message)\n"
    guard let data = line.data(using: .utf8) else { return }

    if FileManager.default.fileExists(atPath: logPath) {
        if let handle = FileHandle(forWritingAtPath: logPath) {
            handle.seekToEndOfFile()
            handle.write(data)
            handle.closeFile()
        }
    } else {
        try? data.write(to: URL(fileURLWithPath: logPath))
    }
}

class AppDelegate: NSObject, NSApplicationDelegate {
    // Registered in applicationWillFinishLaunching, not
    // applicationDidFinishLaunching: the GetURL event that triggers this
    // app's own launch can be delivered before applicationDidFinishLaunching
    // runs, and is lost if the handler isn't registered yet. This is a
    // documented Cocoa requirement for apps launched via a custom URL
    // scheme, not a stylistic choice — verified live on this machine that
    // registering in applicationDidFinishLaunching silently misses every
    // launch-triggering event, while applicationWillFinishLaunching does not.
    func applicationWillFinishLaunching(_ notification: Notification) {
        NSAppleEventManager.shared().setEventHandler(
            self,
            andSelector: #selector(handleGetURL(event:replyEvent:)),
            forEventClass: AEEventClass(kInternetEventClass),
            andEventID: AEEventID(kAEGetURL)
        )
    }

    @objc func handleGetURL(event: NSAppleEventDescriptor, replyEvent: NSAppleEventDescriptor) {
        defer { NSApp.terminate(nil) }

        guard let urlString = event.paramDescriptor(forKeyword: AEKeyword(keyDirectObject))?.stringValue else {
            logFailure("received GetURL event with no URL parameter")
            return
        }

        let launcherPath = NSHomeDirectory() + "/.neat/bin/neat"
        let process = Process()
        process.executableURL = URL(fileURLWithPath: launcherPath)
        process.arguments = ["callback-handler", urlString]
        do {
            try process.run()
        } catch {
            logFailure("failed to launch \(launcherPath): \(error)")
            return
        }
        process.waitUntilExit()
        if process.terminationStatus != 0 {
            logFailure("neat callback-handler exited with status \(process.terminationStatus) for URL \(urlString)")
        }
    }
}

let app = NSApplication.shared
let delegate = AppDelegate()
app.delegate = delegate
app.run()
