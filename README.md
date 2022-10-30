A simple program to export prometheus metrics for all processes on MacOS.

Metrics include cpu + memory % only.

1. Build the binary `go build`
2. Copy the plist to the right location `sudo cp ./io.poyarzun.macos-process-exporter.plist /Library/LaunchDaemons/io.poyarzun.macos-process-exporter.plist`
3. Load the job `sudo launchctl load /Library/LaunchDaemons/io.poyarzun.macos-process-exporter.plist`
4. Start the job `sudo launchctl start /Library/LaunchDaemons/io.poyarzun.macos-process-exporter.plist`