import Foundation

public struct CopiedGPSMetadata: Codable, Equatable, Sendable {
    public let latitude: Double
    public let longitude: Double
    public let altitude: Double?
    public let sourceAssetBaseName: String
    public let sourceFilePath: String
    public let captureDate: String?
    public let gpsSource: String?
    public let gpsMatchMethod: String?
    public let locationSummary: String?
    public let country: String?
    public let province: String?
    public let city: String?
    public let district: String?
    public let copiedDate: Date
    public let rawGPSTags: [String: String]

    public init(
        latitude: Double,
        longitude: Double,
        altitude: Double? = nil,
        sourceAssetBaseName: String,
        sourceFilePath: String,
        captureDate: String? = nil,
        gpsSource: String? = nil,
        gpsMatchMethod: String? = nil,
        locationSummary: String? = nil,
        country: String? = nil,
        province: String? = nil,
        city: String? = nil,
        district: String? = nil,
        copiedDate: Date = Date(),
        rawGPSTags: [String: String] = [:]
    ) {
        self.latitude = latitude
        self.longitude = longitude
        self.altitude = altitude
        self.sourceAssetBaseName = sourceAssetBaseName
        self.sourceFilePath = sourceFilePath
        self.captureDate = captureDate
        self.gpsSource = gpsSource
        self.gpsMatchMethod = gpsMatchMethod
        self.locationSummary = locationSummary
        self.country = country
        self.province = province
        self.city = city
        self.district = district
        self.copiedDate = copiedDate
        self.rawGPSTags = rawGPSTags
    }

    public var formattedDecimal: String {
        let latStr = String(format: "%.6f°", abs(latitude)) + (latitude >= 0 ? " N" : " S")
        let lonStr = String(format: "%.6f°", abs(longitude)) + (longitude >= 0 ? " E" : " W")
        return "\(latStr), \(lonStr)"
    }

    public var formattedDMS: String {
        return "\(toDMS(degrees: latitude, isLat: true)), \(toDMS(degrees: longitude, isLat: false))"
    }

    public var formattedAltitude: String? {
        guard let altitude else { return nil }
        return String(format: "%.1f m", altitude)
    }

    private func toDMS(degrees: Double, isLat: Bool) -> String {
        let absDeg = abs(degrees)
        let d = Int(absDeg)
        let mFloat = (absDeg - Double(d)) * 60.0
        let m = Int(mFloat)
        let s = (mFloat - Double(m)) * 60.0
        let hemi: String
        if isLat {
            hemi = degrees >= 0 ? "N" : "S"
        } else {
            hemi = degrees >= 0 ? "E" : "W"
        }
        return String(format: "%d°%02d'%05.2f\" %@", d, m, s, hemi)
    }

    /// 导出为人类可读的纯文本摘要
    public var plainTextSummary: String {
        var lines: [String] = []
        lines.append("【GPS 坐标】 \(formattedDecimal) (\(formattedDMS))")
        if let alt = formattedAltitude {
            lines.append("【海拔高度】 \(alt)")
        }
        if let loc = locationSummary, !loc.isEmpty {
            lines.append("【地理位置】 \(loc)")
        }
        lines.append("【来源照片】 \(sourceAssetBaseName)")
        if let date = captureDate, !date.isEmpty {
            lines.append("【拍摄时间】 \(date)")
        }
        if !rawGPSTags.isEmpty {
            lines.append("【已采集原始 GPS 标签 (\(rawGPSTags.count) 项)】")
            for (k, v) in sortedGPSTags {
                lines.append("  • \(k): \(v)")
            }
        }
        return lines.joined(separator: "\n")
    }

    /// 按照专业优先级排序的全量 GPS 标签列表
    public var sortedGPSTags: [(key: String, value: String)] {
        let priorityOrder = [
            "GPSVersionID",
            "GPSLatitudeRef",
            "GPSLatitude",
            "GPSLongitudeRef",
            "GPSLongitude",
            "GPSPosition",
            "GPSAltitudeRef",
            "GPSAltitude",
            "GPSTimeStamp",
            "GPSDateStamp",
            "GPSDateTime",
            "GPSSatellites",
            "GPSMapDatum",
            "GPSImgDirectionRef",
            "GPSImgDirection",
            "GPSHPositioningError",
            "GPSSource",
            "GPSMatchMethod"
        ]

        return rawGPSTags.sorted { a, b in
            let idxA = priorityOrder.firstIndex(of: a.key) ?? 999
            let idxB = priorityOrder.firstIndex(of: b.key) ?? 999
            if idxA != idxB {
                return idxA < idxB
            }
            return a.key < b.key
        }
    }
}
