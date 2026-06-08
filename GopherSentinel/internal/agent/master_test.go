package agent

import (
	"context"
	"testing"
	"time"
)

func TestMasterAgent_RecognizeIntent(t *testing.T) {
	master := NewMasterAgent()

	tests := []struct {
		name     string
		query    string
		wantErr  bool
		wantLen  int
	}{
		{
			name:    "log query",
			query:   "查看订单服务的错误日志",
			wantErr: false,
			wantLen: 1,
		},
		{
			name:    "metric query",
			query:   "CPU 使用率是多少",
			wantErr: false,
			wantLen: 2, // matches both metric and log
		},
		{
			name:    "doc query",
			query:   "如何部署服务",
			wantErr: false,
			wantLen: 1,
		},
		{
			name:    "mixed query",
			query:   "服务 CPU 高怎么查日志",
			wantErr: false,
			wantLen: 3, // metric + log + doc agents
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			result, err := master.RecognizeIntent(ctx, tt.query)
			if (err != nil) != tt.wantErr {
				t.Errorf("RecognizeIntent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(result.RequiredAgents) != tt.wantLen {
				t.Errorf("RecognizeIntent() got %d agents, want %d", len(result.RequiredAgents), tt.wantLen)
			}
		})
	}
}

func TestMasterAgent_Handle(t *testing.T) {
	master := NewMasterAgent()

	// 注册测试 Agent
	master.RegisterAgent("doc", &DocAgent{})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := master.Handle(ctx, "如何部署到 K8s")
	if err != nil {
		t.Errorf("Handle() error = %v", err)
		return
	}

	if result == "" {
		t.Error("Handle() returned empty result")
	}
}

func TestReActAgent_Run(t *testing.T) {
	master := NewMasterAgent()
	master.RegisterAgent("doc", &DocAgent{})

	react := NewReActAgent(master)
	react.maxLoops = 3

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := react.Run(ctx, "帮我分析问题")
	if err != nil {
		t.Errorf("Run() error = %v", err)
		return
	}

	if result == "" {
		t.Error("Run() returned empty result")
	}
}

func TestContainsAny(t *testing.T) {
	tests := []struct {
		s        string
		keywords []string
		want     bool
	}{
		{"hello world", []string{"hello", "world"}, true},
		{"hello world", []string{"foo", "bar"}, false},
		{"日志分析", []string{"日志", "log"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			if got := containsAny(tt.s, tt.keywords...); got != tt.want {
				t.Errorf("containsAny() = %v, want %v", got, tt.want)
			}
		})
	}
}
