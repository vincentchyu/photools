package completion

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// GenerateZsh 生成针对 zsh 的全中文智能补全脚本
func GenerateZsh(w io.Writer) {
	script := `#compdef photools ./photools

# 确保 zsh compinit 已初始化
if ! type compdef >/dev/null 2>&1; then
    autoload -Uz compinit && compinit -i
fi

_photools_continents() {
    local -a continents
    continents=(
        'china:中国全境高精/34省市区县全量行政区划/阿勒泰/伊犁/喀纳斯/摄影地标名胜'
        'asia:亚洲全境/日韩/东南亚/南亚及中亚全量城镇扩展包'
        'europe:欧洲全境/阿尔卑斯山脉/冰岛极光区/欧陆主要城市'
        'north-america:北美50州/黄石大峡谷国家公园/加拿大班夫'
        'oceania:大洋洲/澳大利亚大洋路/大堡礁/新西兰南岛冰川峡湾'
        'south-america:南美洲/秘鲁马丘比丘/玻利维亚天空之镜/巴塔哥尼亚'
        'africa:非洲/埃及金字塔/肯尼亚动物大迁徙/纳米布红沙漠'
        'all:一键安装并启用全球全部大洲及中国高精离线数据包'
    )
    _describe -t continents '大洲与国家高精数据包' continents
}

_photools_templates() {
    local -a templates
    templates=(
        '{PREFIX}_{YYYY-MM-DD}_{SEQ}{SUFFIX}:推荐默认规范: 原前缀_日期_序号.扩展名 (如 DSC_2026-08-23_0001.JPG)'
        '{YYYY-MM-DD}_{PREFIX}_{SEQ}{SUFFIX}:日期前置规范: 日期_原前缀_序号 (如 2026-08-23_DSC_0001.JPG)'
        '{YYYYMMDD}_{SEQ}{SUFFIX}:精简紧凑规范: 年月日_序号 (如 20260823_0001.JPG)'
        '{PREFIX}_{YYYY-MM-DD}_{CITY}_{SEQ}{SUFFIX}:地理信息增强: 原前缀_日期_城市_序号'
    )
    _describe -t templates '文件命名模板预设' templates
}

_photools_raw_exts() {
    local -a exts
    exts=(
        'nef,cr3,arw,dng,raf,rw2,orf:全品牌常用主流 RAW 格式'
        'nef:尼康 Nikon RAW'
        'cr3,cr2:佳能 Canon RAW'
        'arw:索尼 Sony RAW'
        'dng:通用 Adobe DNG / 大疆无人机'
        'raf:富士 Fujifilm RAW'
        'rw2:松下 Panasonic RAW'
        'orf:奥林巴斯 Olympus RAW'
    )
    _describe -t exts 'RAW 扩展名列表' exts
}

_photools_geodata() {
    local -a geodata_subcmds
    geodata_subcmds=(
        'list:列出所有支持的大洲数据包及本地安装状态'
        'install:下载并安装指定大洲或中国高精离线数据包'
        'remove:卸载并删除本地已安装的大洲数据包'
        'info:查看当前全局已加载的地理空间坐标点位与索引状态'
        'test:测试并验证指定经纬度坐标的离线逆地理编码匹配 (--debug 开启全链路耗时与拓扑排名)'
    )

    if (( CURRENT == 2 )); then
        _describe -t geodata_subcmds 'geodata 管理指令' geodata_subcmds
    elif (( CURRENT == 3 )); then
        case $words[2] in
            install|get|add|remove|rm|delete)
                _photools_continents
                ;;
        esac
    fi
}

_photools_completion() {
    local -a shells
    shells=(
        'install:自动检测当前 Shell 并一键安装自动补全配置至配置文件'
        'zsh:输出 Zsh 自动补全脚本源码'
        'bash:输出 Bash 自动补全脚本源码'
        'fish:输出 Fish 自动补全脚本源码'
    )
    if (( CURRENT == 2 )); then
        _describe -t shells '补全目标 Shell' shells
    fi
}

_photools() {
    local curcontext="$curcontext" state line
    typeset -A opt_args

    _arguments -C \
        '1: :->command' \
        '*:: :->args'

    case $state in
        command)\
            local -a subcommands
            subcommands=(
                'tui:在交互终端中启动全功能可视化摄影处理工作台 (推荐)'
                'geotag:根据 GPX 轨迹时间轴为照片批量修正写入 GPS 并归档'
                'geocode:[独立能力] 为已有 GPS 坐标的照片批量写入离线中文地名元数据'
                'pipeline:[复合流水线] 自由勾选组合 [GPX修正/逆地理地名/拍摄日归档]'
                'organize-by-date:根据照片 EXIF 拍摄日期整理归档并规范化重命名'
                'restore-test:[测试辅助] 从 Inbox_bak 备份目录一键还原原始照片至 Inbox'
                'geodata:管理全球各大洲及中国高精离线地理数据包 (下载/安装/卸载/测试)'
                'completion:生成或一键安装 Shell Tab 键智能自动补全脚本'
                'help:显示 photools 命令行帮助文档'
            )
            _describe -t subcommands 'photools 功能指令' subcommands
            ;;
        args)
            case $line[1] in
                geodata)
                    _photools_geodata
                    ;;
                completion)
                    _photools_completion
                    ;;
                geotag)
                    _arguments -s \
                        '-base-dir[工作根目录 (包含 Inbox/Processed/Logs)]:目录:_files -/' \
                        '-source-dir[待处理照片输入源目录 (默认 Inbox)]:源目录:_files -/' \
                        '-gpx-dir[GPX 轨迹文件存放目录 (默认 ~/.config/gpx)]:轨迹目录:_files -/' \
                        '-processed-dir[归档分类存放的目标根目录 (默认 Processed)]:目标目录:_files -/' \
                        '-flat[扁平原地模式 (直接扫描并就地处理/保存)]' \
                        '-sidecar-policy[侧车写入策略]:策略:((smart\:"智能分层模式:GPS写RAW/XMP写地名/JPG全嵌(推荐)" sidecar_only\:"纯XMP侧车模式" embed_and_sidecar\:"原图与XMP双写同步" embed_only\:"纯原图内嵌写入"))' \
                        '-sidecar-only[仅生成/修改 {file}.xmp 侧车文件 (等价于 -sidecar-policy=sidecar_only)]' \
                        '-companion-exts[伴随文件扩展名白名单 (如 wav, acr, exf)]:扩展名列表:' \
                        '-in-place[原地规范重命名归档，不建立 YYYY/MMDD 子目录]' \
                        '-geosync[相机与 GPS 轨迹的时间偏差补偿值 (如 +00:00:05 或 -00:01:00)]:时间偏移 (+/-HH:MM:SS):' \
                        '-raw-exts[可识别的 RAW 扩展名列表 (逗号分隔)]:RAW格式:_photools_raw_exts' \
                        '-template[自定义归档文件重命名模板]:命名模板:_photools_templates' \
                        '-geocode[是否启用离线逆地理编码写入城市/省份标签]:选项:((true\:"启用离线地名识别写入" false\:"仅写入GPS经纬度"))' \
                        '-interpolate[启用能力 1.5: 根据前后照片时间推算补全 GPS]' \
                        '-interpolate-window[智能推算最大时间窗口 (如 15m, 30m, 1h)]:时长:' \
                        '-allow-no-gps[无 GPS 坐标时允许跳过地名写入直接归档 (软降级)]' \
                        '-test[开启测试备份模式 (处理前自动备份 Inbox 到 Inbox_bak)]' \
                        '-backup[同 -test，处理前备份原始文件]' \
                        '-backup-dir[自定义测试快照备份的存放目录]:备份目录:_files -/' \
                        '-log-dir[日志与待补清单报告存放目录 (默认 ~/.logs/photools)]:日志目录:_files -/' \
                        '-workers[并发处理照片资产组的 Worker 协程数量]:并发数:'
                    ;;
                geocode)
                    _arguments -s \
                        '-dir[待处理照片输入目录 (默认 Inbox)]:源目录:_files -/' \
                        '-source-dir[待处理照片输入目录]:源目录:_files -/' \
                        '-base-dir[基础工作根目录]:工作目录:_files -/' \
                        '-flat[扁平模式 (在当前目录下直接处理)]' \
                        '-sidecar-policy[侧车写入策略]:策略:((smart\:"智能分层模式:GPS写RAW/XMP写地名/JPG全嵌(推荐)" sidecar_only\:"纯XMP侧车模式" embed_and_sidecar\:"原图与XMP双写同步" embed_only\:"纯原图内嵌写入"))' \
                        '-sidecar-only[仅生成/修改 {file}.xmp 侧车文件 (等价于 -sidecar-policy=sidecar_only)]' \
                        '-companion-exts[伴随文件扩展名白名单 (如 wav, acr, exf)]:扩展名列表:' \
                        '-raw-exts[可识别的 RAW 扩展名列表 (逗号分隔)]:RAW格式:_photools_raw_exts' \
                        '-allow-no-gps[无 GPS 坐标时允许跳过地名写入直接归档]' \
                        '-test[开启测试备份模式 (处理前自动备份到 Inbox_bak)]' \
                        '-backup[同 -test，处理前备份原始文件]' \
                        '-backup-dir[自定义测试快照备份的存放目录]:备份目录:_files -/' \
                        '-log-dir[日志与待补清单报告存放目录 (默认 ~/.logs/photools)]:日志目录:_files -/' \
                        '-workers[并发处理照片资产组的 Worker 协程数量]:并发数:'
                    ;;
                pipeline)
                    _arguments -s \
                        '-base-dir[工作根目录 (包含 Inbox/Processed/Logs)]:工作目录:_files -/' \
                        '-source-dir[待处理的照片原始输入目录 (默认 Inbox)]:源目录:_files -/' \
                        '-gpx-dir[GPX 轨迹文件存放目录 (默认 ~/.config/gpx)]:轨迹目录:_files -/' \
                        '-processed-dir[归档分类存放的目标根目录 (默认 Processed)]:目标目录:_files -/' \
                        '-flat[扁平原地模式 (忽略 Inbox/Processed 分层，直接扫描并就地处理/保存)]' \
                        '-sidecar-policy[侧车写入策略]:策略:((smart\:"智能分层模式:GPS写RAW/XMP写地名/JPG全嵌(推荐)" sidecar_only\:"纯XMP侧车模式" embed_and_sidecar\:"原图与XMP双写同步" embed_only\:"纯原图内嵌写入"))' \
                        '-sidecar-only[仅生成/修改 {file}.xmp 侧车文件 (等价于 -sidecar-policy=sidecar_only)]' \
                        '-companion-exts[伴随文件扩展名白名单 (如 wav, acr, exf)]:扩展名列表:' \
                        '-in-place[原地规范重命名，不建立 YYYY/MMDD 子目录]' \
                        '-geosync[相机与 GPS 轨迹的时间偏差补偿值 (如 +00:00:05)]:时间偏移 (+/-HH:MM:SS):' \
                        '-raw-exts[可识别的 RAW 扩展名列表 (逗号分隔)]:RAW格式:_photools_raw_exts' \
                        '-workers[并发处理照片资产组的 Worker 协程数量]:并发数:' \
                        '-gpx[启用能力 1: GPX 轨迹匹配与 GPS 修正]:选项:((true\:"启用" false\:"跳过"))' \
                        '-interpolate[启用能力 1.5: 根据前后照片时间推算补全 GPS]:选项:((true\:"启用" false\:"跳过"))' \
                        '-interpolate-window[智能推算最大时间窗口 (如 15m, 30m, 1h)]:时长:' \
                        '-geocode[启用能力 2: 逆地理编码写入地名元数据]:选项:((true\:"启用" false\:"跳过"))' \
                        '-allow-no-gps[无 GPS 坐标时允许跳过地名写入直接归档 (软降级)]' \
                        '-archive[启用能力 3: 拍摄日期归档与规范重命名]:选项:((true\:"启用" false\:"跳过"))' \
                        '-test[开启测试备份模式 (处理前自动备份 Inbox 到 Inbox_bak)]' \
                        '-backup[同 -test，处理前备份原始文件]' \
                        '-backup-dir[自定义测试快照备份的存放目录]:备份目录:_files -/' \
                        '-log-dir[日志与待补清单报告存放目录 (默认 ~/.logs/photools)]:日志目录:_files -/'
                    ;;
                organize-by-date)
                    _arguments -s \
                        '-source-dir[待整理的照片原始输入目录]:源目录:_files -/' \
                        '-target-dir[归档分类存放的目标根目录]:目标目录:_files -/' \
                        '-base-dir[基础工作根目录]:工作目录:_files -/' \
                        '-raw-exts[可识别的 RAW 扩展名列表 (逗号分隔)]:RAW格式:_photools_raw_exts' \
                        '-template[自定义归档文件重命名模板]:命名模板:_photools_templates' \
                        '-test[开启测试备份模式 (处理前自动备份)]' \
                        '-backup[同 -test，处理前备份原始文件]' \
                        '-backup-dir[自定义测试快照备份的存放目录]:备份目录:_files -/' \
                        '-log-dir[日志与待补清单报告存放目录 (默认 ~/.logs/photools)]:日志目录:_files -/' \
                        '-geocode[对已有 GPS 照片补充逆地理编码地名]:选项:((true\:"启用地名补充" false\:"不修改地名"))'
                    ;;
                restore-test|restore-backup)
                    _arguments -s \
                        '-base-dir[工作根目录 (包含 Inbox/Inbox_bak/Processed)]:工作目录:_files -/' \
                        '-backup-dir[备份源目录 (默认 <base-dir>/Inbox_bak)]:备份源目录:_files -/' \
                        '-target-dir[还原目标目录 (默认 <base-dir>/Inbox)]:还原目标目录:_files -/' \
                        '-clean[还原后是否同时清空 Processed 归档目录以恢复完全初始状态]'
                    ;;
            esac
            ;;
    esac
}

# 显式绑定命令、当前路径二进制及任意绝对/相对路径结尾
compdef _photools photools
compdef _photools ./photools
compdef _photools -p '*/photools'
`
	fmt.Fprint(w, script)
}

// GenerateBash 生成针对 bash 的全中文智能补全脚本
func GenerateBash(w io.Writer) {
	script := `_photools_complete() {
    local cur prev words cword
    _init_completion || return

    local commands="tui geotag geocode pipeline organize-by-date restore-test geodata completion help"
    local geodata_cmds="list install remove info test"
    local continents="china asia europe north-america oceania south-america africa all"
    local completion_subcmds="install zsh bash fish"

    if [[ $cword -eq 1 ]]; then
        COMPREPLY=( $(compgen -W "${commands}" -- "${cur}") )
        return 0
    fi

    case "${words[1]}" in
        geodata)
            if [[ $cword -eq 2 ]]; then
                COMPREPLY=( $(compgen -W "${geodata_cmds}" -- "${cur}") )
            elif [[ $cword -eq 3 ]]; then
                case "${words[2]}" in
                    install|get|add|remove|rm|delete)
                        COMPREPLY=( $(compgen -W "${continents}" -- "${cur}") )
                        ;;
                esac
            fi
            ;;
        completion)
            if [[ $cword -eq 2 ]]; then
                COMPREPLY=( $(compgen -W "${completion_subcmds}" -- "${cur}") )
            fi
            ;;
        geotag)
            if [[ "$cur" == -* ]]; then
                COMPREPLY=( $(compgen -W "-base-dir -source-dir -gpx-dir -processed-dir -flat -sidecar-policy -sidecar-only -companion-exts -in-place -geosync -raw-exts -template -geocode -interpolate -interpolate-window -allow-no-gps -workers -test -backup -backup-dir" -- "${cur}") )
            fi
            ;;
        geocode)
            if [[ "$cur" == -* ]]; then
                COMPREPLY=( $(compgen -W "-dir -source-dir -base-dir -flat -sidecar-policy -sidecar-only -companion-exts -raw-exts -allow-no-gps -workers -test -backup -backup-dir" -- "${cur}") )
            fi
            ;;
        pipeline)
            if [[ "$cur" == -* ]]; then
                COMPREPLY=( $(compgen -W "-base-dir -source-dir -gpx-dir -processed-dir -flat -sidecar-policy -sidecar-only -companion-exts -in-place -geosync -raw-exts -workers -gpx -interpolate -interpolate-window -geocode -allow-no-gps -archive -test -backup -backup-dir" -- "${cur}") )
            fi
            ;;
        organize-by-date)
            if [[ "$cur" == -* ]]; then
                COMPREPLY=( $(compgen -W "-source-dir -target-dir -base-dir -raw-exts -template -geocode -test -backup -backup-dir" -- "${cur}") )
            fi
            ;;
        restore-test|restore-backup)
            if [[ "$cur" == -* ]]; then
                COMPREPLY=( $(compgen -W "-base-dir -backup-dir -target-dir -clean" -- "${cur}") )
            fi
            ;;
    esac
}

complete -F _photools_complete photools ./photools
`
	fmt.Fprint(w, script)
}

// GenerateFish 生成针对 fish 的全中文智能补全脚本
func GenerateFish(w io.Writer) {
	script := `complete -c photools -f
complete -c photools -n "__fish_use_subcommand" -a tui -d "在交互终端中启动可视化摄影处理工作台"
complete -c photools -n "__fish_use_subcommand" -a geotag -d "根据 GPX 轨迹为照片批量修正写入 GPS 并归档"
complete -c photools -n "__fish_use_subcommand" -a geocode -d "为已有 GPS 坐标的照片批量写入离线中文地名元数据"
complete -c photools -n "__fish_use_subcommand" -a pipeline -d "自由勾选组合流水线 [GPX修正/逆地理地名/拍摄日归档]"
complete -c photools -n "__fish_use_subcommand" -a organize-by-date -d "根据拍摄日期整理归档并规范化重命名"
complete -c photools -n "__fish_use_subcommand" -a restore-test -d "从 Inbox_bak 备份目录一键还原原始照片至 Inbox"
complete -c photools -n "__fish_use_subcommand" -a geodata -d "管理全球各大洲及中国高精离线地理数据包"
complete -c photools -n "__fish_use_subcommand" -a completion -d "生成或一键安装 Shell 自动补全"
complete -c photools -n "__fish_use_subcommand" -a help -d "显示 photools 命令行帮助"

# geodata 子命令
complete -c photools -n "__fish_seen_subcommand_from geodata" -a list -d "列出所有可用大洲及本地安装状态"
complete -c photools -n "__fish_seen_subcommand_from geodata" -a install -d "下载并安装指定大洲离线数据包"
complete -c photools -n "__fish_seen_subcommand_from geodata" -a remove -d "卸载并移除指定大洲离线数据包"
complete -c photools -n "__fish_seen_subcommand_from geodata" -a info -d "查看当前地理空间索引状态与点位总数"
complete -c photools -n "__fish_seen_subcommand_from geodata" -a test -d "测试并验证指定经纬度坐标的逆地理编码"

# geodata 大洲选项
complete -c photools -n "__fish_seen_subcommand_from install remove get add delete rm" -a china -d "中国全境高精/34省市区县全量行政区划与摄影地标名胜"
complete -c photools -n "__fish_seen_subcommand_from install remove get add delete rm" -a asia -d "亚洲全境城镇扩展包"
complete -c photools -n "__fish_seen_subcommand_from install remove get add delete rm" -a europe -d "欧洲全境摄影胜地与城市"
complete -c photools -n "__fish_seen_subcommand_from install remove get add delete rm" -a north-america -d "北美50州与国家公园"
complete -c photools -n "__fish_seen_subcommand_from install remove get add delete rm" -a oceania -d "大洋洲/澳大利亚与新西兰"
complete -c photools -n "__fish_seen_subcommand_from install remove get add delete rm" -a south-america -d "南美洲/天空之镜与巴塔哥尼亚"
complete -c photools -n "__fish_seen_subcommand_from install remove get add delete rm" -a africa -d "非洲/动物大迁徙与金字塔"
complete -c photools -n "__fish_seen_subcommand_from install remove get add delete rm" -a all -d "一键安装全部大洲与中国高精包"

# geotag 选项
complete -c photools -n "__fish_seen_subcommand_from geotag" -l base-dir -d "工作根目录 (包含 Inbox/Processed/Logs)"
complete -c photools -n "__fish_seen_subcommand_from geotag" -l source-dir -d "待处理照片输入源目录"
complete -c photools -n "__fish_seen_subcommand_from geotag" -l gpx-dir -d "GPX 轨迹文件存放目录 (默认 ~/.config/gpx)"
complete -c photools -n "__fish_seen_subcommand_from geotag" -l processed-dir -d "归档目标根目录"
complete -c photools -n "__fish_seen_subcommand_from geotag" -l flat -d "扁平原地模式 (直接扫描并就地处理/保存)"
complete -c photools -n "__fish_seen_subcommand_from geotag" -l sidecar-policy -d "侧车写入策略 (smart/sidecar_only/embed_and_sidecar/embed_only)"
complete -c photools -n "__fish_seen_subcommand_from geotag" -l sidecar-only -d "仅生成/修改 {file}.xmp 侧车文件，不修改原始 RAW/JPG 文件"
complete -c photools -n "__fish_seen_subcommand_from geotag" -l companion-exts -d "伴随文件扩展名白名单 (如 wav, acr, exf)"
complete -c photools -n "__fish_seen_subcommand_from geotag" -l in-place -d "原地规范重命名归档，不建立 YYYY/MMDD 子目录"
complete -c photools -n "__fish_seen_subcommand_from geotag" -l geosync -d "相机与 GPS 轨迹的时间偏差补偿值"
complete -c photools -n "__fish_seen_subcommand_from geotag" -l interpolate -d "启用根据前后照片时间推算补全 GPS"
complete -c photools -n "__fish_seen_subcommand_from geotag" -l interpolate-window -d "智能推算最大时间窗口 (如 15m, 30m, 1h)"
complete -c photools -n "__fish_seen_subcommand_from geotag" -l allow-no-gps -d "无 GPS 照片允许跳过地名写入直接归档"
complete -c photools -n "__fish_seen_subcommand_from geotag" -l raw-exts -d "可识别的 RAW 扩展名列表"
complete -c photools -n "__fish_seen_subcommand_from geotag" -l workers -d "并发处理 Worker 协程数量"
complete -c photools -n "__fish_seen_subcommand_from geotag" -l test -d "开启测试备份模式"
complete -c photools -n "__fish_seen_subcommand_from geotag" -l backup -d "同 -test，处理前备份原始文件"

# geocode 选项
complete -c photools -n "__fish_seen_subcommand_from geocode" -l dir -d "待处理照片输入目录"
complete -c photools -n "__fish_seen_subcommand_from geocode" -l source-dir -d "待处理照片输入目录"
complete -c photools -n "__fish_seen_subcommand_from geocode" -l base-dir -d "基础工作根目录"
complete -c photools -n "__fish_seen_subcommand_from geocode" -l flat -d "扁平模式 (在当前目录下直接处理)"
complete -c photools -n "__fish_seen_subcommand_from geocode" -l sidecar-policy -d "侧车写入策略 (read_only/sidecar_only/embed_and_sidecar/embed_only)"
complete -c photools -n "__fish_seen_subcommand_from geocode" -l sidecar-only -d "仅生成/修改 {file}.xmp 侧车文件，不修改原始 RAW/JPG 文件"
complete -c photools -n "__fish_seen_subcommand_from geocode" -l companion-exts -d "伴随文件扩展名白名单 (如 wav, acr, exf)"
complete -c photools -n "__fish_seen_subcommand_from geocode" -l raw-exts -d "可识别的 RAW 扩展名列表"
complete -c photools -n "__fish_seen_subcommand_from geocode" -l allow-no-gps -d "无 GPS 照片允许跳过地名写入直接归档"
complete -c photools -n "__fish_seen_subcommand_from geocode" -l workers -d "并发处理 Worker 协程数量"

# pipeline 选项
complete -c photools -n "__fish_seen_subcommand_from pipeline" -l base-dir -d "工作根目录"
complete -c photools -n "__fish_seen_subcommand_from pipeline" -l source-dir -d "待处理照片原始输入目录"
complete -c photools -n "__fish_seen_subcommand_from pipeline" -l gpx-dir -d "GPX 轨迹文件存放目录 (默认 ~/.config/gpx)"
complete -c photools -n "__fish_seen_subcommand_from pipeline" -l processed-dir -d "归档目标根目录"
complete -c photools -n "__fish_seen_subcommand_from pipeline" -l flat -d "扁平原地模式 (直接扫描并就地处理/保存)"
complete -c photools -n "__fish_seen_subcommand_from pipeline" -l sidecar-policy -d "侧车写入策略 (smart/sidecar_only/embed_and_sidecar/embed_only)"
complete -c photools -n "__fish_seen_subcommand_from pipeline" -l sidecar-only -d "仅生成/修改 {file}.xmp 侧车文件，不修改原始 RAW/JPG 文件"
complete -c photools -n "__fish_seen_subcommand_from pipeline" -l companion-exts -d "伴随文件扩展名白名单 (如 wav, acr, exf)"
complete -c photools -n "__fish_seen_subcommand_from pipeline" -l in-place -d "原地规范重命名，不建立 YYYY/MMDD 子目录"
complete -c photools -n "__fish_seen_subcommand_from geotag" -l geosync -d "相机与 GPS 轨迹的时间偏差补偿值"
complete -c photools -n "__fish_seen_subcommand_from geotag" -l interpolate -d "启用根据前后照片时间推算补全 GPS"
complete -c photools -n "__fish_seen_subcommand_from geotag" -l interpolate-window -d "智能推算最大时间窗口 (如 15m, 30m, 1h)"
complete -c photools -n "__fish_seen_subcommand_from geotag" -l allow-no-gps -d "无 GPS 照片允许跳过地名写入直接归档"
complete -c photools -n "__fish_seen_subcommand_from geotag" -l raw-exts -d "可识别的 RAW 扩展名列表"
complete -c photools -n "__fish_seen_subcommand_from geotag" -l workers -d "并发处理 Worker 协程数量"
complete -c photools -n "__fish_seen_subcommand_from geotag" -l test -d "开启测试备份模式"
complete -c photools -n "__fish_seen_subcommand_from geotag" -l backup -d "同 -test，处理前备份原始文件"

# geocode 选项
complete -c photools -n "__fish_seen_subcommand_from geocode" -l dir -d "待处理照片输入目录"
complete -c photools -n "__fish_seen_subcommand_from geocode" -l source-dir -d "待处理照片输入目录"
complete -c photools -n "__fish_seen_subcommand_from geocode" -l base-dir -d "基础工作根目录"
complete -c photools -n "__fish_seen_subcommand_from geocode" -l flat -d "扁平模式 (在当前目录下直接处理)"
complete -c photools -n "__fish_seen_subcommand_from geocode" -l raw-exts -d "可识别的 RAW 扩展名列表"
complete -c photools -n "__fish_seen_subcommand_from geocode" -l allow-no-gps -d "无 GPS 照片允许跳过地名写入直接归档"
complete -c photools -n "__fish_seen_subcommand_from geocode" -l workers -d "并发处理 Worker 协程数量"

# pipeline 选项
complete -c photools -n "__fish_seen_subcommand_from pipeline" -l base-dir -d "工作根目录"
complete -c photools -n "__fish_seen_subcommand_from pipeline" -l source-dir -d "待处理照片原始输入目录"
complete -c photools -n "__fish_seen_subcommand_from pipeline" -l gpx-dir -d "GPX 轨迹文件存放目录 (默认 ~/.config/gpx)"
complete -c photools -n "__fish_seen_subcommand_from pipeline" -l processed-dir -d "归档目标根目录"
complete -c photools -n "__fish_seen_subcommand_from pipeline" -l flat -d "扁平原地模式 (直接扫描并就地处理/保存)"
complete -c photools -n "__fish_seen_subcommand_from pipeline" -l in-place -d "原地规范重命名，不建立 YYYY/MMDD 子目录"
complete -c photools -n "__fish_seen_subcommand_from pipeline" -l geosync -d "相机与 GPS 时间偏差补偿值"
complete -c photools -n "__fish_seen_subcommand_from pipeline" -l gpx -d "启用能力 1: GPX 轨迹匹配"
complete -c photools -n "__fish_seen_subcommand_from pipeline" -l interpolate -d "启用能力 1.5: GPS 智能时间插值"
complete -c photools -n "__fish_seen_subcommand_from pipeline" -l interpolate-window -d "智能推算最大时间窗口"
complete -c photools -n "__fish_seen_subcommand_from pipeline" -l geocode -d "启用能力 2: 逆地理编码写入"
complete -c photools -n "__fish_seen_subcommand_from pipeline" -l allow-no-gps -d "无 GPS 照片允许跳过地名写入直接归档"
complete -c photools -n "__fish_seen_subcommand_from pipeline" -l archive -d "启用能力 3: 拍摄日期归档与规范重命名"
complete -c photools -n "__fish_seen_subcommand_from pipeline" -l raw-exts -d "可识别的 RAW 扩展名列表"
complete -c photools -n "__fish_seen_subcommand_from pipeline" -l workers -d "并发处理 Worker 数量"

# organize-by-date 选项
complete -c photools -n "__fish_seen_subcommand_from organize-by-date" -l source-dir -d "待整理的照片源目录"
complete -c photools -n "__fish_seen_subcommand_from organize-by-date" -l target-dir -d "归档分类存放目标根目录"
complete -c photools -n "__fish_seen_subcommand_from organize-by-date" -l base-dir -d "基础工作根目录"

# restore-test 选项
complete -c photools -n "__fish_seen_subcommand_from restore-test" -l base-dir -d "工作根目录"
complete -c photools -n "__fish_seen_subcommand_from restore-test" -l backup-dir -d "备份源目录"
complete -c photools -n "__fish_seen_subcommand_from restore-test" -l target-dir -d "还原目标目录"
complete -c photools -n "__fish_seen_subcommand_from restore-test" -l clean -d "还原后是否同时清空 Processed 目录"

# completion 子命令
complete -c photools -n "__fish_seen_subcommand_from completion" -a install -d "一键安装自动补全配置到 Shell 配置文件"
complete -c photools -n "__fish_seen_subcommand_from completion" -a zsh -d "输出 Zsh 补全脚本"
complete -c photools -n "__fish_seen_subcommand_from completion" -a bash -d "输出 Bash 补全脚本"
complete -c photools -n "__fish_seen_subcommand_from completion" -a fish -d "输出 Fish 补全脚本"
`
	fmt.Fprint(w, script)
}

// InstallShellCompletion 自动为当前用户的 Shell (~/.zshrc 或 ~/.bashrc) 配置补全脚本
func InstallShellCompletion() (string, error) {
	shell := os.Getenv("SHELL")
	home := os.Getenv("HOME")
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return "", err
		}
	}

	targetFile := filepath.Join(home, ".zshrc")
	shellType := "zsh"
	if strings.Contains(shell, "bash") {
		targetFile = filepath.Join(home, ".bashrc")
		shellType = "bash"
	}

	compDir := filepath.Join(home, ".config", "photools", "completions")
	if err := os.MkdirAll(compDir, 0o755); err != nil {
		return "", err
	}

	scriptPath := filepath.Join(compDir, "photools."+shellType)
	f, err := os.Create(scriptPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if shellType == "zsh" {
		GenerateZsh(f)
	} else {
		GenerateBash(f)
	}

	sourceLine := fmt.Sprintf(`[ -f "%s" ] && source "%s"`, scriptPath, scriptPath)

	rcContent, _ := os.ReadFile(targetFile)
	if !strings.Contains(string(rcContent), scriptPath) {
		appended := string(rcContent)
		if len(appended) > 0 && !strings.HasSuffix(appended, "\n") {
			appended += "\n"
		}
		appended += "\n# photools Shell Auto-Completion\n" + sourceLine + "\n"
		if err := os.WriteFile(targetFile, []byte(appended), 0o644); err != nil {
			return "", err
		}
	}

	return fmt.Sprintf("已成功为 %s 安装全指令中文自动补全配置至 %s！\n请执行 'source %s' 或新开终端窗口后生效。", shellType, targetFile, targetFile), nil
}
