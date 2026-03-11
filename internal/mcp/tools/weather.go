package tools

import "context"

type WeatherHandler struct{}

func (WeatherHandler) Name() string {
	return ToolWeather
}

func (WeatherHandler) Call(_ context.Context, call ToolCall) (map[string]any, error) {
	args := call.Arguments
	random := randomByArgs(args)
	return map[string]any{
		"city":         city(readText(args, "city")),
		"date":         readText(args, "date"),
		"temperatureC": random.NextInt(28) + 5,
		"humidity":     35 + random.NextInt(55),
		"windLevel":    1 + random.NextInt(7),
		"condition":    weatherConditions[random.NextInt(len(weatherConditions))],
		"mockTag":      "幂等随机数据",
	}, nil
}
