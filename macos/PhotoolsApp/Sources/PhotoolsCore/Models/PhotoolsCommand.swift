import Foundation

public struct GeotagRunOptions: Equatable, Sendable {
    public var baseDirectory: String
    public var geosync: String
    public var rawExtensions: String
    public var workers: Int

    public init(baseDirectory: String, geosync: String, rawExtensions: String, workers: Int) {
        self.baseDirectory = baseDirectory
        self.geosync = geosync
        self.rawExtensions = rawExtensions
        self.workers = workers
    }
}

public struct PipelineRunOptions: Equatable, Sendable {
    public var baseDirectory: String
    public var sourceDirectory: String
    public var gpxDirectory: String
    public var processedDirectory: String
    public var flatMode: Bool
    public var sidecarPolicy: String
    public var companionExtensions: String
    public var inPlace: Bool
    public var geosync: String
    public var rawExtensions: String
    public var workers: Int
    public var enableGPX: Bool
    public var enableInterpolate: Bool
    public var interpolateWindow: String
    public var enableGeocode: Bool
    public var allowNoGPS: Bool
    public var enableArchive: Bool
    public var testBackup: Bool
    public var backupDirectory: String

    public init(
        baseDirectory: String = "",
        sourceDirectory: String = "",
        gpxDirectory: String = "",
        processedDirectory: String = "",
        flatMode: Bool = false,
        sidecarPolicy: String = "smart",
        companionExtensions: String = "wav,acr,exf",
        inPlace: Bool = false,
        geosync: String = "0",
        rawExtensions: String = "nef,cr3,arw,dng,raf,rw2,orf",
        workers: Int = ProcessInfo.processInfo.processorCount,
        enableGPX: Bool = true,
        enableInterpolate: Bool = false,
        interpolateWindow: String = "15m",
        enableGeocode: Bool = true,
        allowNoGPS: Bool = false,
        enableArchive: Bool = true,
        testBackup: Bool = false,
        backupDirectory: String = ""
    ) {
        self.baseDirectory = baseDirectory
        self.sourceDirectory = sourceDirectory
        self.gpxDirectory = gpxDirectory
        self.processedDirectory = processedDirectory
        self.flatMode = flatMode
        self.sidecarPolicy = sidecarPolicy
        self.companionExtensions = companionExtensions
        self.inPlace = inPlace
        self.geosync = geosync
        self.rawExtensions = rawExtensions
        self.workers = workers
        self.enableGPX = enableGPX
        self.enableInterpolate = enableInterpolate
        self.interpolateWindow = interpolateWindow
        self.enableGeocode = enableGeocode
        self.allowNoGPS = allowNoGPS
        self.enableArchive = enableArchive
        self.testBackup = testBackup
        self.backupDirectory = backupDirectory
    }
}

public struct PhotoolsCommand: Equatable, Sendable {
    public let executablePath: String
    public let arguments: [String]

    public init(executablePath: String, arguments: [String]) {
        self.executablePath = executablePath
        self.arguments = arguments
    }

    public static func geotag(executablePath: String, options: GeotagRunOptions) -> PhotoolsCommand {
        PhotoolsCommand(
            executablePath: executablePath,
            arguments: [
                "geotag",
                "-base-dir", options.baseDirectory,
                "-geosync", options.geosync,
                "-raw-exts", options.rawExtensions,
                "-workers", String(options.workers)
            ]
        )
    }

    public static func pipeline(executablePath: String, options: PipelineRunOptions) -> PhotoolsCommand {
        var args = ["pipeline"]
        if !options.baseDirectory.isEmpty {
            args.append(contentsOf: ["-base-dir", options.baseDirectory])
        }
        if !options.sourceDirectory.isEmpty {
            args.append(contentsOf: ["-source-dir", options.sourceDirectory])
        }
        if !options.gpxDirectory.isEmpty {
            args.append(contentsOf: ["-gpx-dir", options.gpxDirectory])
        }
        if !options.processedDirectory.isEmpty {
            args.append(contentsOf: ["-processed-dir", options.processedDirectory])
        }
        if !options.geosync.isEmpty && options.geosync != "0" {
            args.append(contentsOf: ["-geosync", options.geosync])
        } else if !options.geosync.isEmpty {
            args.append(contentsOf: ["-geosync", "0"])
        }
        if !options.rawExtensions.isEmpty {
            args.append(contentsOf: ["-raw-exts", options.rawExtensions])
        }
        if options.workers > 0 {
            args.append(contentsOf: ["-workers", String(options.workers)])
        }
        if !options.interpolateWindow.isEmpty && options.interpolateWindow != "15m" {
            args.append(contentsOf: ["-interpolate-window", options.interpolateWindow])
        } else if !options.interpolateWindow.isEmpty {
            args.append(contentsOf: ["-interpolate-window", "15m"])
        }
        if options.testBackup && !options.backupDirectory.isEmpty {
            args.append(contentsOf: ["-backup-dir", options.backupDirectory])
        }

        if options.flatMode {
            args.append("-flat")
        }
        if !options.sidecarPolicy.isEmpty {
            args.append(contentsOf: ["-sidecar-policy", options.sidecarPolicy])
        }
        if !options.companionExtensions.isEmpty {
            args.append(contentsOf: ["-companion-exts", options.companionExtensions])
        }
        if options.inPlace {
            args.append("-in-place")
        }
        if options.enableGPX {
            args.append("-gpx")
        }
        if options.enableInterpolate {
            args.append("-interpolate")
        }
        if options.enableGeocode {
            args.append("-geocode")
        }
        if options.allowNoGPS {
            args.append("-allow-no-gps")
        }
        if options.enableArchive {
            args.append("-archive")
        }
        if options.testBackup {
            args.append("-test")
        }

        return PhotoolsCommand(executablePath: executablePath, arguments: args)
    }

    public static func restoreTest(
        executablePath: String,
        baseDirectory: String,
        backupDir: String = "",
        targetDir: String = "",
        cleanProcessed: Bool = false
    ) -> PhotoolsCommand {
        var args = ["restore-test"]
        if !baseDirectory.isEmpty {
            args.append(contentsOf: ["-base-dir", baseDirectory])
        }
        if !backupDir.isEmpty {
            args.append(contentsOf: ["-backup-dir", backupDir])
        }
        if !targetDir.isEmpty {
            args.append(contentsOf: ["-target-dir", targetDir])
        }
        if cleanProcessed {
            args.append("-clean")
        }
        return PhotoolsCommand(executablePath: executablePath, arguments: args)
    }

    public static func createBackup(
        executablePath: String,
        baseDirectory: String,
        sourceDir: String = "",
        backupDir: String = ""
    ) -> PhotoolsCommand {
        var args = ["backup"]
        if !baseDirectory.isEmpty {
            args.append(contentsOf: ["-base-dir", baseDirectory])
        }
        if !sourceDir.isEmpty {
            args.append(contentsOf: ["-source-dir", sourceDir])
        }
        if !backupDir.isEmpty {
            args.append(contentsOf: ["-backup-dir", backupDir])
        }
        return PhotoolsCommand(executablePath: executablePath, arguments: args)
    }

    public static func geodataList(executablePath: String) -> PhotoolsCommand {
        PhotoolsCommand(executablePath: executablePath, arguments: ["geodata", "list"])
    }

    public static func geodataInstall(executablePath: String, target: String) -> PhotoolsCommand {
        PhotoolsCommand(executablePath: executablePath, arguments: ["geodata", "install", target])
    }

    public static func geodataRemove(executablePath: String, target: String) -> PhotoolsCommand {
        PhotoolsCommand(executablePath: executablePath, arguments: ["geodata", "remove", target])
    }

    public static func geodataInfo(executablePath: String) -> PhotoolsCommand {
        PhotoolsCommand(executablePath: executablePath, arguments: ["geodata", "info"])
    }

    public static func geodataTest(
        executablePath: String,
        latitude: Double,
        longitude: Double,
        altitude: Double = 0.0,
        debug: Bool = false
    ) -> PhotoolsCommand {
        let latStr = (latitude == Double(Int(latitude))) ? "\(Int(latitude))" : "\(latitude)"
        let lonStr = (longitude == Double(Int(longitude))) ? "\(Int(longitude))" : "\(longitude)"
        let altStr = "\(Int(altitude))"
        var args = ["geodata", "test", latStr, lonStr, altStr]
        if debug {
            args.append("--debug")
        }
        return PhotoolsCommand(executablePath: executablePath, arguments: args)
    }
}
