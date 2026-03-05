package tools

import (
	"context"
	"fmt"
	"time"
)

type TodoHandler struct{}

func (TodoHandler) Name() string {
	return ToolTodo
}

func (TodoHandler) Call(_ context.Context, args map[string]any) (map[string]any, error) {
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
	}, nil
}
