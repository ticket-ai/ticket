package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/pkoukk/tiktoken-go"
	"github.com/ticket-ai/ticket/pkg/budget"
)


type ModelConfig struct {
	InputCostPer1K  int64

	OutputCostPer1K int64

	MaxOutputTokens int
}

var ModelConfigs = map[string]ModelConfig{
	"gpt-4":            {InputCostPer1K: 30, OutputCostPer1K: 60, MaxOutputTokens: 4096},
	"gpt-4-32k":        {InputCostPer1K: 60, OutputCostPer1K: 120, MaxOutputTokens: 32768},
	"gpt-3.5-turbo":    {InputCostPer1K: 1, OutputCostPer1K: 2, MaxOutputTokens: 4096},
	"gpt-3.5-turbo-16k":{InputCostPer1K: 2, OutputCostPer1K: 4, MaxOutputTokens: 16384},

	"claude-2":         {InputCostPer1K: 8, OutputCostPer1K: 24, MaxOutputTokens: 4096},
	"claude-instant":   {InputCostPer1K: 1, OutputCostPer1K: 3, MaxOutputTokens: 4096},
}

type LLMManager struct {
	bm *budget.BudgetManager
}

func NewLLMManager(bm *budget.BudgetManager) *LLMManager {
	return &LLMManager{bm: bm}
}

type Request struct {
	Model     string    `json:"model"`
	Messages  []Message `json:"messages,omitempty"`
	Prompt    string    `json:"prompt,omitempty"`
	MaxTokens int       `json:"max_tokens,omitempty"`
}

type Message struct {
	Role    string `json:"role"`

	Content string `json:"content"`
}

var encoder *tiktoken.Encoding

func init() {
	var err error
	encoder, err = tiktoken.GetEncoding("cl100k_base")
	
	if err != nil {
		panic(fmt.Errorf("tokenizer init: %w", err))
	}
}



func (lm *LLMManager) ProcessRequest(w http.ResponseWriter, r *http.Request) error {

	var buf bytes.Buffer
	tee := io.TeeReader(r.Body, &buf)
	body, err := io.ReadAll(tee)

	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	r.Body = io.NopCloser(&buf)

	var req Request

	if err := json.Unmarshal(body, &req); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	cfg, ok := ModelConfigs[req.Model]

	if !ok {
		return fmt.Errorf("unknown model: %s", req.Model)
	}

	outTokens := req.MaxTokens

	if outTokens <= 0 || outTokens > cfg.MaxOutputTokens {
		outTokens = cfg.MaxOutputTokens

		w.Header().Set("X-Max-Output-Tokens", strconv.Itoa(cfg.MaxOutputTokens))
	}

	var inputTokens int

	if len(req.Messages) > 0 {
		for _, m := range req.Messages {
			tokens := encoder.Encode(m.Content, nil)
			inputTokens += len(tokens)
		}

	} else {
		tokens := encoder.Encode(req.Prompt, nil)
		inputTokens = len(tokens)
	}

	inputCost := (int64(inputTokens)*cfg.InputCostPer1K + 999) / 1000

	outputCost := (int64(outTokens)*cfg.OutputCostPer1K + 999) / 1000

	totalCost := inputCost + outputCost

	uuid := r.Header.Get("X-User-ID")


	if uuid == "" {
		return fmt.Errorf("missing X-User-ID header")
	}

	if err := lm.bm.DeductFromBalance(uuid, totalCost); err != nil {
		return fmt.Errorf("billing failed: %w", err)
	}

	if err := lm.bm.TrackModelUsage(uuid, req.Model, totalCost); err != nil {
		log.Printf("track analytics: %v", err)
	}

	return nil
}
