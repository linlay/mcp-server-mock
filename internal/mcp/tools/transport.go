package tools

import (
	"context"
	"fmt"
	"strings"
)

type TransportHandler struct{}

func (TransportHandler) Name() string {
	return ToolTransport
}

func (TransportHandler) Call(_ context.Context, args map[string]any) (map[string]any, error) {
	random := randomByArgs(args)
	rawType := readText(args, "type")
	travelType := "航班"
	if strings.EqualFold(rawType, "train") || rawType == "高铁" {
		travelType = "高铁"
	}

	fromCity := city(readText(args, "fromCity"))
	toCity := city(readText(args, "toCity"))
	date := readText(args, "date")

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
	}, nil
}
