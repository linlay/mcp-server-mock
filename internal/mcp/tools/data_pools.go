package tools

import "regexp"

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

type sensitiveRule struct {
	label   string
	pattern *regexp.Regexp
}
