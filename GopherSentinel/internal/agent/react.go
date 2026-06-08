package agent

import (
	"context"
	"fmt"
)

// ReActAgent 实现 ReAct (Reason + Act) 模式的 Agent
type ReActAgent struct {
	name        string
	maxLoops    int
	timeout     int
	masterAgent *MasterAgent
}

// NewReActAgent 创建 ReAct Agent
func NewReActAgent(master *MasterAgent) *ReActAgent {
	return &ReActAgent{
		name:        "react",
		maxLoops:    10,
		timeout:     30,
		masterAgent: master,
	}
}

// Thought 思考步骤
type Thought struct {
	Step       int
	Reasoning  string
	Action     string
	Observation string
}

// Run 执行 ReAct 循环
func (r *ReActAgent) Run(ctx context.Context, query string) (string, error) {
	history := []Thought{}

	for i := 0; i < r.maxLoops; i++ {
		// 1. Think - 思考下一步
		thought := r.think(ctx, query, history)

		// 2. 检查是否应该结束
		if thought.Action == "finish" {
			return thought.Observation, nil
		}

		// 3. Act - 执行行动
		obs, err := r.act(ctx, thought.Action)
		thought.Observation = obs

		if err != nil {
			// 反思错误原因
			thought = r.reflect(ctx, thought, err)
			history = append(history, thought)
			continue
		}

		history = append(history, thought)

		// 4. 检查是否超时或达到最大循环
		if len(history) >= r.maxLoops {
			break
		}
	}

	// 超过最大循环次数，返回当前最佳结果
	return r.giveUp(history), nil
}

// think 思考步骤
func (r *ReActAgent) think(ctx context.Context, query string, history []Thought) Thought {
	thought := Thought{
		Step: len(history) + 1,
	}

	// 简单的基于规则的思考
	// 实际项目中可以使用 LLM 进行更智能的思考
	if len(history) == 0 {
		// 第一步：识别意图
		thought.Reasoning = "First step: recognize intent"
		thought.Action = "recognize_intent"
	} else if len(history) < 3 {
		// 前几步：收集信息
		thought.Reasoning = "Collect more information"
		thought.Action = "collect_info"
	} else {
		// 后续：综合分析
		thought.Reasoning = "Analyze collected information"
		thought.Action = "analyze"
	}

	return thought
}

// act 执行行动
func (r *ReActAgent) act(ctx context.Context, action string) (string, error) {
	switch action {
	case "recognize_intent":
		// 调用 Master Agent 进行意图识别
		intent, err := r.masterAgent.RecognizeIntent(ctx, "query")
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Intent recognized: %v", intent.RequiredAgents), nil

	case "collect_info":
		return "Information collected", nil

	case "analyze":
		return "Analysis complete", nil

	case "finish":
		return "Task completed", nil

	default:
		return "", fmt.Errorf("unknown action: %s", action)
	}
}

// reflect 反思
func (r *ReActAgent) reflect(ctx context.Context, thought Thought, err error) Thought {
	thought.Reasoning = fmt.Sprintf("Error occurred: %v. Try a different approach.", err)
	thought.Action = "collect_info"
	return thought
}

// giveUp 放弃并返回结果
func (r *ReActAgent) giveUp(history []Thought) string {
	if len(history) == 0 {
		return "Unable to complete task"
	}

	summary := "Summary of reasoning:\n"
	for _, t := range history {
		summary += fmt.Sprintf("Step %d: %s -> %s\n", t.Step, t.Reasoning, t.Action)
	}
	return summary
}
