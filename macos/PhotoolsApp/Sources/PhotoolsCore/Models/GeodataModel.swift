import Foundation

public struct GeodataContinentPack: Identifiable, Equatable, Sendable {
    public let id: String
    public let code: String
    public let nameZH: String
    public let description: String
    public let isInstalled: Bool
    public let pointCount: Int
    public let sizeMB: Double

    public init(
        code: String,
        nameZH: String,
        description: String,
        isInstalled: Bool,
        pointCount: Int,
        sizeMB: Double
    ) {
        self.id = code
        self.code = code
        self.nameZH = nameZH
        self.description = description
        self.isInstalled = isInstalled
        self.pointCount = pointCount
        self.sizeMB = sizeMB
    }
}

public struct GeodataCandidatePoint: Identifiable, Equatable, Sendable {
    public let id: String
    public let rank: Int
    public let name: String
    public let nameZH: String
    public let featureDesc: String
    public let locationHierarchy: String
    public let distanceKm: Double
    public let lat: Double
    public let lon: Double
    public let elevation: Int
    public let geonameID: Int
    public let source: String

    public init(
        rank: Int,
        name: String,
        nameZH: String,
        featureDesc: String,
        locationHierarchy: String,
        distanceKm: Double,
        lat: Double,
        lon: Double,
        elevation: Int = 0,
        geonameID: Int = 0,
        source: String = ""
    ) {
        self.id = "\(rank)_\(geonameID)_\(lat)_\(lon)"
        self.rank = rank
        self.name = name
        self.nameZH = nameZH
        self.featureDesc = featureDesc
        self.locationHierarchy = locationHierarchy
        self.distanceKm = distanceKm
        self.lat = lat
        self.lon = lon
        self.elevation = elevation
        self.geonameID = geonameID
        self.source = source
    }
}

public struct GeodataLookupResult: Equatable, Sendable {
    public let country: String
    public let countryCode: String
    public let province: String
    public let city: String
    public let district: String
    public let timezone: String
    public let elevation: Int
    public let distanceKm: Double
    public let source: String
    public let formattedSummary: String
    public let debugText: String
    public let candidates: [GeodataCandidatePoint]

    public init(
        country: String = "",
        countryCode: String = "",
        province: String = "",
        city: String = "",
        district: String = "",
        timezone: String = "",
        elevation: Int = 0,
        distanceKm: Double = 0,
        source: String = "",
        formattedSummary: String = "",
        debugText: String = "",
        candidates: [GeodataCandidatePoint] = []
    ) {
        self.country = country
        self.countryCode = countryCode
        self.province = province
        self.city = city
        self.district = district
        self.timezone = timezone
        self.elevation = elevation
        self.distanceKm = distanceKm
        self.source = source
        self.formattedSummary = formattedSummary
        self.debugText = debugText
        self.candidates = candidates
    }
}

public struct GeodataParser: Sendable {
    public static func parseListOutput(_ output: String) -> [GeodataContinentPack] {
        var packs: [GeodataContinentPack] = []
        let lines = output.components(separatedBy: .newlines)

        var currentCode: String?
        var currentNameZH: String?
        var currentInstalled: Bool = false
        var currentPointCount: Int = 0
        var currentSizeMB: Double = 0.0
        var currentDesc: String = ""

        for line in lines {
            let trimmed = line.trimmingCharacters(in: .whitespaces)
            if trimmed.hasPrefix("•") {
                // 如果已有前一个包，先收集
                if let code = currentCode, let name = currentNameZH {
                    packs.append(
                        GeodataContinentPack(
                            code: code,
                            nameZH: name,
                            description: currentDesc,
                            isInstalled: currentInstalled,
                            pointCount: currentPointCount,
                            sizeMB: currentSizeMB
                        )
                    )
                }

                currentDesc = ""
                currentPointCount = 0
                currentSizeMB = 0.0

                // 格式如: • china           [中国离线高精地名库 (含港澳台)]: ✅ 已安装 (715000 点位, 42.5 MB)
                let parts = trimmed.dropFirst().trimmingCharacters(in: .whitespaces)
                let scanner = Scanner(string: parts)

                if let code = scanner.scanUpToString("[")?.trimmingCharacters(in: .whitespaces) {
                    currentCode = code
                }
                _ = scanner.scanString("[")
                if let name = scanner.scanUpToString("]")?.trimmingCharacters(in: .whitespaces) {
                    currentNameZH = name
                }
                _ = scanner.scanString("]:")
                let rest = scanner.scanUpToString("\n") ?? parts

                currentInstalled = rest.contains("已安装") || rest.contains("✅")
                if currentInstalled {
                    // 提取点位
                    if let ptRange = rest.range(of: "点位") {
                        let prefix = rest[..<ptRange.lowerBound]
                        if let lParen = prefix.range(of: "(", options: .backwards) {
                            let numStr = prefix[lParen.upperBound...].trimmingCharacters(in: .whitespaces)
                            currentPointCount = Int(numStr) ?? 0
                        }
                    }
                    // 提取大小
                    if let mbRange = rest.range(of: "MB") {
                        let prefix = rest[..<mbRange.lowerBound]
                        if let comma = prefix.range(of: ",", options: .backwards) {
                            let sizeStr = prefix[comma.upperBound...].trimmingCharacters(in: .whitespaces)
                            currentSizeMB = Double(sizeStr) ?? 0.0
                        }
                    }
                }
            } else if trimmed.hasPrefix("└─") {
                currentDesc = String(trimmed.dropFirst(2)).trimmingCharacters(in: .whitespaces)
            }
        }

        if let code = currentCode, let name = currentNameZH {
            packs.append(
                GeodataContinentPack(
                    code: code,
                    nameZH: name,
                    description: currentDesc,
                    isInstalled: currentInstalled,
                    pointCount: currentPointCount,
                    sizeMB: currentSizeMB
                )
            )
        }

        return packs
    }

    public static func parseListJSON(_ jsonString: String) -> [GeodataContinentPack] {
        guard let data = jsonString.data(using: .utf8),
              let array = try? JSONSerialization.jsonObject(with: data) as? [[String: Any]] else {
            return []
        }

        var results: [GeodataContinentPack] = []
        for item in array {
            let meta = item["meta"] as? [String: Any]
            guard let code = meta?["code"] as? String ?? item["code"] as? String else {
                continue
            }
            let name = meta?["name_zh"] as? String ?? item["name"] as? String ?? item["name_zh"] as? String ?? code
            let desc = meta?["description"] as? String ?? item["description"] as? String ?? ""
            let isInstalled = item["installed"] as? Bool ?? item["is_installed"] as? Bool ?? false
            let ptCount = item["points"] as? Int ?? item["point_count"] as? Int ?? meta?["approx_points"] as? Int ?? 0

            var sizeMB: Double = 0.0
            if let bytes = item["file_size"] as? Int64 {
                sizeMB = Double(bytes) / (1024.0 * 1024.0)
            } else if let bytes = item["file_size"] as? Int {
                sizeMB = Double(bytes) / (1024.0 * 1024.0)
            } else if let bytes = item["file_size"] as? Double {
                sizeMB = bytes / (1024.0 * 1024.0)
            } else if let mb = item["size_mb"] as? Double {
                sizeMB = mb
            }

            results.append(
                GeodataContinentPack(
                    code: code,
                    nameZH: name,
                    description: desc,
                    isInstalled: isInstalled,
                    pointCount: ptCount,
                    sizeMB: sizeMB
                )
            )
        }
        return results
    }

    public static func parseLookupOutput(_ output: String) -> GeodataLookupResult? {
        guard output.contains("【逆地理编码匹配结果】") else {
            return nil
        }

        var country = ""
        var countryCode = ""
        var province = ""
        var city = ""
        var district = ""
        var timezone = ""
        var elevation = 0
        var distanceKm = 0.0
        var source = ""
        var formattedSummary = ""

        let lines = output.components(separatedBy: .newlines)
        for line in lines {
            let trimmed = line.trimmingCharacters(in: .whitespaces)
            if trimmed.hasPrefix("• 国家:") {
                let val = trimmed.replacingOccurrences(of: "• 国家:", with: "").trimmingCharacters(in: .whitespaces)
                // 如 "中国 (CN)"
                if let lParen = val.range(of: "("), let rParen = val.range(of: ")") {
                    country = String(val[..<lParen.lowerBound]).trimmingCharacters(in: .whitespaces)
                    countryCode = String(val[lParen.upperBound..<rParen.lowerBound]).trimmingCharacters(in: .whitespaces)
                } else {
                    country = val
                }
            } else if trimmed.hasPrefix("• 省份/州:") {
                province = trimmed.replacingOccurrences(of: "• 省份/州:", with: "").trimmingCharacters(in: .whitespaces)
            } else if trimmed.hasPrefix("• 城市/地区:") {
                city = trimmed.replacingOccurrences(of: "• 城市/地区:", with: "").trimmingCharacters(in: .whitespaces)
            } else if trimmed.hasPrefix("• 区县/POI:") {
                district = trimmed.replacingOccurrences(of: "• 区县/POI:", with: "").trimmingCharacters(in: .whitespaces)
            } else if trimmed.hasPrefix("• IANA时区:") {
                timezone = trimmed.replacingOccurrences(of: "• IANA时区:", with: "").trimmingCharacters(in: .whitespaces)
            } else if trimmed.hasPrefix("• 地形海拔:") {
                let numStr = trimmed.replacingOccurrences(of: "• 地形海拔:", with: "").replacingOccurrences(of: "米", with: "").trimmingCharacters(in: .whitespaces)
                elevation = Int(numStr) ?? 0
            } else if trimmed.hasPrefix("• 物理距离:") {
                let numStr = trimmed.replacingOccurrences(of: "• 物理距离:", with: "").replacingOccurrences(of: "km", with: "").trimmingCharacters(in: .whitespaces)
                distanceKm = Double(numStr) ?? 0.0
            } else if trimmed.hasPrefix("• 数据源:") {
                source = trimmed.replacingOccurrences(of: "• 数据源:", with: "").trimmingCharacters(in: .whitespaces)
            } else if trimmed.hasPrefix("• 规范全称:") {
                formattedSummary = trimmed.replacingOccurrences(of: "• 规范全称:", with: "").trimmingCharacters(in: .whitespaces)
            }
        }

        return GeodataLookupResult(
            country: country,
            countryCode: countryCode,
            province: province,
            city: city,
            district: district,
            timezone: timezone,
            elevation: elevation,
            distanceKm: distanceKm,
            source: source,
            formattedSummary: formattedSummary,
            debugText: output
        )
    }
}
