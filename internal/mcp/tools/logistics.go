package tools

import (
	"context"
	"time"
)

type LogisticsHandler struct{}

func (LogisticsHandler) Name() string {
	return ToolLogistics
}

func (LogisticsHandler) Call(_ context.Context, args map[string]any) (map[string]any, error) {
	random := randomByArgs(args)
	trackingNo := readText(args, "trackingNo")
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
	}, nil
}
