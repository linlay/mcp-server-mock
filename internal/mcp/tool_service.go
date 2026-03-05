package mcp

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"mcp-server-mock-go/internal/observability"
)

const (
	toolWeather   = "mock.weather.query"
	toolLogistics = "mock.logistics.status"
	toolRunbook   = "mock.ops.runbook.generate"
	toolSensitive = "mock.sensitive-data.detect"
	toolTodo      = "mock.todo.tasks.list"
	toolTransport = "mock.transport.schedule.query"
)

var (
	weatherConditions = []string{"晴", "多云", "晴间多云", "小雨", "雷阵雨", "雾", "小雪"}

	logisticsCarriers = []string{"SF Express", "YTO Express", "ZTO Express", "JD Logistics", "EMS"}
	logisticsStatuses = []string{"ORDER_CONFIRMED", "IN_TRANSIT", "OUT_FOR_DELIVERY", "DELIVERED"}
	logisticsNodes    = []string{
		"Parcel information received",
		"Departed sorting center",
		"Arrived destination city",
		"Courier is delivering",
		"Delivered and signed",
	}

	riskLevels = []string{"low", "medium", "high"}

	todoPool = []string{
		"整理需求文档", "回复客户邮件", "同步项目进展", "代码评审", "准备周会汇报",
		"修复线上缺陷", "更新测试用例", "部署预发环境", "跟进物流异常", "预订差旅行程",
	}
	todoPriorities = []string{"高", "中", "低"}
	todoStatuses   = []string{"待处理", "进行中", "已完成"}

	flightNumbers = []string{"MU5137", "CA1502", "CZ3948", "HO1256", "ZH2871"}
	trainNumbers  = []string{"G102", "G356", "D2285", "G7311", "C2610"}
	flightStatus  = []string{"计划中", "值机中", "正点", "延误"}
	trainStatus   = []string{"计划中", "检票中", "正点", "晚点"}

	cityMap = map[string]string{
		"shanghai":      "上海",
		"beijing":       "北京",
		"guangzhou":     "广州",
		"shenzhen":      "深圳",
		"hangzhou":      "杭州",
		"chengdu":       "成都",
		"wuhan":         "武汉",
		"nanjing":       "南京",
		"xian":          "西安",
		"xi'an":         "西安",
		"chongqing":     "重庆",
		"tianjin":       "天津",
		"suzhou":        "苏州",
		"hong kong":     "香港",
		"taipei":        "台北",
		"singapore":     "新加坡",
		"tokyo":         "东京",
		"osaka":         "大阪",
		"seoul":         "首尔",
		"new york":      "纽约",
		"los angeles":   "洛杉矶",
		"san francisco": "旧金山",
		"london":        "伦敦",
		"paris":         "巴黎",
		"berlin":        "柏林",
		"sydney":        "悉尼",
		"melbourne":     "墨尔本",
		"dubai":         "迪拜",
	}

	sensitiveRules = []sensitiveRule{
		{label: "手机号", pattern: regexp.MustCompile(`(?:^|\D)1[3-9]\d{9}(?:$|\D)`)},
		{label: "邮箱", pattern: regexp.MustCompile(`(?i)\b[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}\b`)},
		{label: "身份证号", pattern: regexp.MustCompile(`(?i)\b[1-9]\d{5}(18|19|20)\d{2}(0[1-9]|1[0-2])(0[1-9]|[12]\d|3[01])\d{3}[0-9X]\b`)},
		{label: "银行卡号", pattern: regexp.MustCompile(`(?:^|\D)(?:\d[ -]?){16,19}(?:$|\D)`)},
		{label: "OpenAI Key", pattern: regexp.MustCompile(`\bsk-[A-Za-z0-9_-]{16,}\b`)},
		{label: "AWS Access Key", pattern: regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`)},
	}
)

// ToolService implements tool listing and tool calls.
type ToolService struct {
	repository *ToolSpecRepository
	logger     *observability.Logger
}

func NewToolService(repository *ToolSpecRepository, logger *observability.Logger) *ToolService {
	return &ToolService{repository: repository, logger: logger}
}

func (s *ToolService) ListTools() []map[string]any {
	if s.repository == nil {
		return []map[string]any{}
	}
	return s.repository.ListTools()
}

func (s *ToolService) CallTool(rawToolName string, args map[string]any) map[string]any {
	start := time.Now()
	toolName := canonicalToolName(rawToolName)
	safeArgs := args
	if safeArgs == nil {
		safeArgs = map[string]any{}
	}
	if s.logger != nil {
		s.logger.LogToolRequest(rawToolName, toolName, safeArgs)
	}

	if toolName == "" {
		errResult := errorResult("unknown tool: " + strings.TrimSpace(rawToolName))
		if s.logger != nil {
			s.logger.LogToolError(rawToolName, toolName, time.Since(start), toString(errResult["error"]))
		}
		return errResult
	}

	var structured map[string]any
	switch toolName {
	case toolWeather:
		structured = weather(safeArgs)
	case toolLogistics:
		structured = logistics(safeArgs)
	case toolRunbook:
		structured = opsRunbook(safeArgs)
	case toolSensitive:
		structured = sensitiveDetect(safeArgs)
	case toolTodo:
		structured = todoTasks(safeArgs)
	case toolTransport:
		structured = transportSchedule(safeArgs)
	default:
		structured = nil
	}

	if structured == nil {
		errResult := errorResult("tool implementation missing: " + toolName)
		if s.logger != nil {
			s.logger.LogToolError(rawToolName, toolName, time.Since(start), toString(errResult["error"]))
		}
		return errResult
	}

	structuredText, _ := json.Marshal(structured)
	result := map[string]any{
		"structuredContent": structured,
		"content": []map[string]any{{
			"type": "text",
			"text": string(structuredText),
		}},
		"isError": false,
	}
	if s.logger != nil {
		s.logger.LogToolResponse(toolName, result, time.Since(start))
	}
	return result
}

func weather(args map[string]any) map[string]any {
	cityName := city(readTextOrDefault(args, "city", "shanghai"))
	date := readTextOrDefault(args, "date", "1970-01-01")
	random := randomByArgs(args)

	return map[string]any{
		"city":         cityName,
		"date":         date,
		"temperatureC": random.NextInt(28) + 5,
		"humidity":     35 + random.NextInt(55),
		"windLevel":    1 + random.NextInt(7),
		"condition":    weatherConditions[random.NextInt(len(weatherConditions))],
		"mockTag":      "幂等随机数据",
	}
}

func logistics(args map[string]any) map[string]any {
	random := randomByArgs(args)
	trackingNo := readText(args, "trackingNo")
	if trackingNo == "" {
		trackingNo = fmt.Sprintf("MOCK%d", 100000+random.NextInt(900000))
	}

	carrier := readText(args, "carrier")
	if carrier == "" {
		carrier = logisticsCarriers[random.NextInt(len(logisticsCarriers))]
	}

	statusIndex := random.NextInt(len(logisticsStatuses))
	status := logisticsStatuses[statusIndex]
	nodeIndex := statusIndex + 1
	if nodeIndex >= len(logisticsNodes) {
		nodeIndex = len(logisticsNodes) - 1
	}

	etaDate := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC).
		AddDate(0, 0, 1+random.NextInt(5)).Format("2006-01-02")
	updatedAt := time.Date(2026, time.January, 1, 8, 0, 0, 0, time.UTC).
		Add(time.Duration(random.NextInt(120)) * time.Hour).
		Add(time.Duration(random.NextInt(60)) * time.Minute).
		Format("2006-01-02 15:04:05")

	return map[string]any{
		"trackingNo":  trackingNo,
		"carrier":     carrier,
		"status":      status,
		"currentNode": logisticsNodes[nodeIndex],
		"etaDate":     etaDate,
		"updatedAt":   updatedAt,
		"mockTag":     "idempotent-random-json",
	}
}

func opsRunbook(args map[string]any) map[string]any {
	message := toString(orValue(args, "message", orValue(args, "query", "")))
	cityName := toString(orValue(args, "city", "Shanghai"))
	random := randomByArgs(args)
	command := "ls -la"
	if random.NextBool() {
		command = "df -h"
	}

	return map[string]any{
		"message":            message,
		"city":               cityName,
		"riskLevel":          riskLevels[random.NextInt(len(riskLevels))],
		"recommendedCommand": command,
		"steps": []string{
			"检查系统负载与磁盘利用率",
			"确认业务实例状态",
			"输出巡检摘要",
		},
		"mockTag": "idempotent-random-json",
	}
}

func sensitiveDetect(args map[string]any) map[string]any {
	text := firstNonBlank(
		readAny(args, "text"),
		readAny(args, "content"),
		readAny(args, "message"),
		readAny(args, "query"),
		readAny(args, "document"),
		readAny(args, "input"),
	)

	if strings.TrimSpace(text) == "" {
		return map[string]any{
			"hasSensitiveData": false,
			"result":           "没有敏感数据",
			"description":      "未检测到可分析文本。",
		}
	}

	for _, rule := range sensitiveRules {
		if rule.pattern.MatchString(text) {
			return map[string]any{
				"hasSensitiveData": true,
				"result":           "有敏感数据",
				"description":      "检测到疑似" + rule.label + "信息，建议脱敏后再传输。",
			}
		}
	}

	return map[string]any{
		"hasSensitiveData": false,
		"result":           "没有敏感数据",
		"description":      "未发现明显敏感字段特征。",
	}
}

func todoTasks(args map[string]any) map[string]any {
	random := randomByArgs(args)
	owner := readText(args, "owner")
	if owner == "" {
		owner = "当前用户"
	}

	total := 3 + random.NextInt(4)
	tasks := make([]map[string]any, 0, total)
	for i := 0; i < total; i++ {
		tasks = append(tasks, map[string]any{
			"id":       fmt.Sprintf("TASK-%d", 100+i),
			"title":    todoPool[(i+random.NextInt(len(todoPool)))%len(todoPool)],
			"priority": todoPriorities[random.NextInt(len(todoPriorities))],
			"status":   todoStatuses[random.NextInt(len(todoStatuses))],
			"dueDate": time.Date(2026, time.February, 13, 0, 0, 0, 0, time.UTC).
				AddDate(0, 0, 1+random.NextInt(7)).Format("2006-01-02"),
		})
	}

	return map[string]any{
		"owner":   owner,
		"total":   total,
		"tasks":   tasks,
		"mockTag": "幂等随机数据",
	}
}

func transportSchedule(args map[string]any) map[string]any {
	random := randomByArgs(args)
	rawType := readText(args, "type")
	travelType := "航班"
	if strings.EqualFold(rawType, "train") || rawType == "高铁" {
		travelType = "高铁"
	}

	fromCity := city(readTextOrDefault(args, "fromCity", "shanghai"))
	toCity := city(readTextOrDefault(args, "toCity", "beijing"))
	date := readTextOrDefault(args, "date", "2026-02-13")

	departureHour := 6 + random.NextInt(14)
	departureMinute := 30
	if random.NextBool() {
		departureMinute = 0
	}

	durationMinutes := 180 + random.NextInt(240)
	if travelType == "航班" {
		durationMinutes = 90 + random.NextInt(150)
	}
	arrivalTotal := departureHour*60 + departureMinute + durationMinutes
	arrivalHour := (arrivalTotal / 60) % 24
	arrivalMinute := arrivalTotal % 60

	number := trainNumbers[random.NextInt(len(trainNumbers))]
	status := trainStatus[random.NextInt(len(trainStatus))]
	gateOrPlatform := fmt.Sprintf("%d 站台", 1+random.NextInt(16))
	if travelType == "航班" {
		number = flightNumbers[random.NextInt(len(flightNumbers))]
		status = flightStatus[random.NextInt(len(flightStatus))]
		gateOrPlatform = fmt.Sprintf("T%d-%d", 1+random.NextInt(2), 10+random.NextInt(20))
	}

	return map[string]any{
		"travelType":     travelType,
		"number":         number,
		"fromCity":       fromCity,
		"toCity":         toCity,
		"date":           date,
		"departureTime":  formatHM(departureHour, departureMinute),
		"arrivalTime":    formatHM(arrivalHour, arrivalMinute),
		"status":         status,
		"gateOrPlatform": gateOrPlatform,
		"mockTag":        "幂等随机数据",
	}
}

func errorResult(message string) map[string]any {
	if strings.TrimSpace(message) == "" {
		message = "unknown error"
	}
	return map[string]any{
		"isError": true,
		"error":   message,
		"content": []map[string]any{{
			"type": "text",
			"text": message,
		}},
	}
}

func canonicalToolName(rawToolName string) string {
	normalized := strings.ToLower(strings.TrimSpace(rawToolName))
	if normalized == "" {
		return ""
	}
	supported := map[string]struct{}{
		toolWeather:   {},
		toolLogistics: {},
		toolRunbook:   {},
		toolSensitive: {},
		toolTodo:      {},
		toolTransport: {},
	}
	if _, ok := supported[normalized]; ok {
		return normalized
	}
	return ""
}

func randomByArgs(args map[string]any) *javaRandom {
	seedBase := javaMapString(args)
	var seed int64
	for _, b := range []byte(seedBase) {
		seed = seed*31 + int64(b)
	}
	return newJavaRandom(seed)
}

func javaMapString(args map[string]any) string {
	if len(args) == 0 {
		return "{}"
	}
	keys := make([]string, 0, len(args))
	for key := range args {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, javaValueString(args[key])))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func javaValueString(value any) string {
	switch typed := value.(type) {
	case nil:
		return "null"
	case map[string]any:
		return javaMapString(typed)
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			parts = append(parts, javaValueString(item))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	default:
		return fmt.Sprint(typed)
	}
}

func readText(args map[string]any, key string) string {
	if args == nil {
		return ""
	}
	value, exists := args[key]
	if !exists || value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func readTextOrDefault(args map[string]any, key, fallback string) string {
	value := readText(args, key)
	if value == "" {
		return fallback
	}
	return value
}

func readAny(args map[string]any, key string) string {
	if args == nil {
		return ""
	}
	value, exists := args[key]
	if !exists || value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func formatHM(hour, minute int) string {
	return fmt.Sprintf("%02d:%02d", hour, minute)
}

func city(value string) string {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return "上海"
	}
	mapped, ok := cityMap[strings.ToLower(normalized)]
	if ok {
		return mapped
	}
	return normalized
}

func orValue(args map[string]any, key string, fallback any) any {
	if args == nil {
		return fallback
	}
	value, exists := args[key]
	if !exists {
		return fallback
	}
	return value
}

type sensitiveRule struct {
	label   string
	pattern *regexp.Regexp
}

type javaRandom struct {
	seed uint64
}

const (
	javaMultiplier = 0x5DEECE66D
	javaAddend     = 0xB
	javaMask       = (1 << 48) - 1
)

func newJavaRandom(seed int64) *javaRandom {
	return &javaRandom{seed: (uint64(seed) ^ javaMultiplier) & javaMask}
}

func (r *javaRandom) next(bits uint) int32 {
	r.seed = (r.seed*javaMultiplier + javaAddend) & javaMask
	return int32(r.seed >> (48 - bits))
}

func (r *javaRandom) NextInt(bound int) int {
	if bound <= 0 {
		return 0
	}
	if bound&(bound-1) == 0 {
		return int((int64(bound) * int64(r.next(31))) >> 31)
	}
	for {
		bits := int(r.next(31))
		val := bits % bound
		if bits-val+(bound-1) >= 0 {
			return val
		}
	}
}

func (r *javaRandom) NextBool() bool {
	return r.next(1) != 0
}
