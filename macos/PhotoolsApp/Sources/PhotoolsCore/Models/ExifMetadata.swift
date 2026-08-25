import Foundation

public struct ExifTagItem: Identifiable, Hashable, Sendable {
    public var id: String { "\(group):\(tag)" }
    public let group: String
    public let tag: String
    public let value: String

    public init(group: String, tag: String, value: String) {
        self.group = group
        self.tag = tag
        self.value = value
    }
}

public struct ExifMetadata: Sendable {
    public let filePath: String
    public let fileName: String
    public let fileSize: String
    public let fileModifyDate: String

    // 核心拍摄参数
    public let cameraMake: String?
    public let cameraModel: String?
    public let lensModel: String?
    public let dateTimeOriginal: String?
    public let exposureTime: String?
    public let fNumber: String?
    public let iso: String?
    public let focalLength: String?
    public let exposureProgram: String?

    // GPS 坐标与海拔
    public let latitude: Double?
    public let longitude: Double?
    public let altitude: Double?
    public let gpsPosition: String?

    // IPTC / XMP 中文地名与元数据
    public let country: String?
    public let province: String?
    public let city: String?
    public let district: String?
    public let title: String?
    public let description: String?

    // 全部原始标签列表（按 Group 分组）
    public let rawTags: [ExifTagItem]

    public init(
        filePath: String,
        fileName: String,
        fileSize: String,
        fileModifyDate: String,
        cameraMake: String? = nil,
        cameraModel: String? = nil,
        lensModel: String? = nil,
        dateTimeOriginal: String? = nil,
        exposureTime: String? = nil,
        fNumber: String? = nil,
        iso: String? = nil,
        focalLength: String? = nil,
        exposureProgram: String? = nil,
        latitude: Double? = nil,
        longitude: Double? = nil,
        altitude: Double? = nil,
        gpsPosition: String? = nil,
        country: String? = nil,
        province: String? = nil,
        city: String? = nil,
        district: String? = nil,
        title: String? = nil,
        description: String? = nil,
        rawTags: [ExifTagItem] = []
    ) {
        self.filePath = filePath
        self.fileName = fileName
        self.fileSize = fileSize
        self.fileModifyDate = fileModifyDate
        self.cameraMake = cameraMake
        self.cameraModel = cameraModel
        self.lensModel = lensModel
        self.dateTimeOriginal = dateTimeOriginal
        self.exposureTime = exposureTime
        self.fNumber = fNumber
        self.iso = iso
        self.focalLength = focalLength
        self.exposureProgram = exposureProgram
        self.latitude = latitude
        self.longitude = longitude
        self.altitude = altitude
        self.gpsPosition = gpsPosition
        self.country = country
        self.province = province
        self.city = city
        self.district = district
        self.title = title
        self.description = description
        self.rawTags = rawTags
    }

    public var hasGPS: Bool {
        latitude != nil && longitude != nil
    }

    public var cameraSummary: String {
        if let model = cameraModel, !model.isEmpty {
            if let make = cameraMake, !make.isEmpty, !model.lowercased().contains(make.lowercased()) {
                return "\(make) \(model)"
            }
            return model
        }
        if let make = cameraMake, !make.isEmpty {
            return make
        }
        return "未知相机型号"
    }

    public var exposureSummary: String {
        var parts: [String] = []
        if let exp = exposureTime, !exp.isEmpty {
            parts.append(exp.hasSuffix("s") ? exp : "\(exp)s")
        }
        if let fn = fNumber, !fn.isEmpty {
            parts.append(fn.starts(with: "f/") || fn.starts(with: "F/") ? fn : "f/\(fn)")
        }
        if let isoVal = iso, !isoVal.isEmpty {
            parts.append("ISO \(isoVal)")
        }
        if let fl = focalLength, !fl.isEmpty {
            parts.append(fl.hasSuffix("mm") ? fl : "\(fl)mm")
        }
        return parts.isEmpty ? "未记录曝光参数" : parts.joined(separator: " · ")
    }

    public var locationSummary: String? {
        var parts: [String] = []
        if let c = country, !c.isEmpty { parts.append(c) }
        if let p = province, !p.isEmpty { parts.append(p) }
        if let ct = city, !ct.isEmpty && ct != province { parts.append(ct) }
        if let d = district, !d.isEmpty { parts.append(d) }
        if parts.isEmpty {
            return nil
        }
        return parts.joined(separator: " · ")
    }

    public static func parse(from dict: [String: Any], fallbackPath: String) -> ExifMetadata {
        func strVal(_ keys: String...) -> String? {
            for k in keys {
                if let val = dict[k] {
                    let s = "\(val)".trimmingCharacters(in: .whitespacesAndNewlines)
                    if !s.isEmpty && s != "<nil>" && s != "0" && s != "undef" && s != "-" {
                        return s
                    }
                }
            }
            return nil
        }

        func doubleVal(_ keys: String...) -> Double? {
            for k in keys {
                if let val = dict[k] {
                    if let d = val as? Double { return d }
                    if let s = val as? String, let d = Double(s) { return d }
                    if let num = val as? NSNumber { return num.doubleValue }
                }
            }
            return nil
        }

        let filePath = strVal("file_path", "SourceFile") ?? fallbackPath
        let fileName = strVal("file_name") ?? (filePath as NSString).lastPathComponent
        let fileSize = strVal("file_size") ?? ""
        let modDate = strVal("file_modify_date") ?? ""

        let make = strVal("camera_make", "EXIF:Make", "Make")
        let model = strVal("camera_model", "EXIF:Model", "Model")
        let lens = strVal("lens_model", "EXIF:LensModel", "XMP:LensModel", "Composite:LensSpec", "LensModel")
        let date = strVal("date_time_original", "EXIF:DateTimeOriginal", "XMP:DateTimeOriginal", "DateTimeOriginal")
        let exposure = strVal("exposure_time", "EXIF:ExposureTime", "ExposureTime")
        let fNumber = strVal("f_number", "EXIF:FNumber", "FNumber")
        let iso = strVal("iso", "EXIF:ISO", "ISO")
        let focal = strVal("focal_length", "EXIF:FocalLength", "FocalLength")
        let program = strVal("exposure_program", "EXIF:ExposureProgram", "ExposureProgram")

        let lat = doubleVal("latitude", "Composite:GPSLatitude", "GPS:GPSLatitude", "GPSLatitude")
        let lon = doubleVal("longitude", "Composite:GPSLongitude", "GPS:GPSLongitude", "GPSLongitude")
        let alt = doubleVal("altitude", "Composite:GPSAltitude", "GPS:GPSAltitude", "GPSAltitude")
        let pos = strVal("gps_position", "Composite:GPSPosition", "GPSPosition")

        let country = strVal("country", "XMP:Country", "IPTC:Country-PrimaryLocationName", "Country")
        let province = strVal("province", "XMP:State", "IPTC:Province-State", "State", "Province")
        let city = strVal("city", "XMP:City", "IPTC:City", "City")
        let district = strVal("district", "XMP:Location", "IPTC:Sub-location", "District", "Location")
        let title = strVal("title", "XMP:Title", "IPTC:ObjectName", "Title")
        let desc = strVal("description", "XMP:Description", "IPTC:Caption-Abstract", "Description")

        var tags: [ExifTagItem] = []
        if let rawTagList = dict["raw_tags"] as? [[String: Any]] {
            for item in rawTagList {
                let g = item["group"] as? String ?? "General"
                let t = item["tag"] as? String ?? ""
                let v = item["value"] as? String ?? ""
                if !t.isEmpty {
                    tags.append(ExifTagItem(group: g, tag: t, value: v))
                }
            }
        } else {
            for (k, v) in dict {
                if k == "SourceFile" || k == "file_path" || k == "raw_tags" { continue }
                let parts = k.split(separator: ":", maxSplits: 1)
                let group = parts.count > 1 ? String(parts[0]) : "General"
                let tag = parts.count > 1 ? String(parts[1]) : k
                tags.append(ExifTagItem(group: group, tag: tag, value: "\(v)"))
            }
        }
        tags.sort { ($0.group, $0.tag) < ($1.group, $1.tag) }

        return ExifMetadata(
            filePath: filePath,
            fileName: fileName,
            fileSize: fileSize,
            fileModifyDate: modDate,
            cameraMake: make,
            cameraModel: model,
            lensModel: lens,
            dateTimeOriginal: date,
            exposureTime: exposure,
            fNumber: fNumber,
            iso: iso,
            focalLength: focal,
            exposureProgram: program,
            latitude: lat,
            longitude: lon,
            altitude: alt,
            gpsPosition: pos,
            country: country,
            province: province,
            city: city,
            district: district,
            title: title,
            description: desc,
            rawTags: tags
        )
    }
}
