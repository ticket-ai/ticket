package llm

import (
	"encoding/json"
	"net/http"
	"time"
)

type MockResponse struct {
	Text         string  `json:"text"`
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	Cost         float64 `json:"cost"`
}

type MockServer struct {
	server *http.Server
	port   string
}

func NewMockServer(port string) *MockServer {
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	mock := &MockServer{
		server: server,
		port:   port,
	}

	mux.HandleFunc("/v1/chat/completions", mock.handleChatCompletion)
	mux.HandleFunc("/v1/completions", mock.handleCompletion)

	return mock
}

func (m *MockServer) Start() error {
	return m.server.ListenAndServe()
}

func (m *MockServer) Stop() error {
	return m.server.Close()
}

func (m *MockServer) GetURL() string {
	return "http://localhost:" + m.port
}

func (m *MockServer) handleChatCompletion(w http.ResponseWriter, r *http.Request) {

	time.Sleep(100 * time.Millisecond)

	model := r.Header.Get("X-Model")
	if model == "" {
		model = "gpt-3.5-turbo"
	}

	inputTokens := 50
	outputTokens := 100
	cost := calculateMockCost(model, inputTokens, outputTokens)

	response := MockResponse{
		Text:         "This is a mock response",
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		Cost:         cost,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Input-Tokens", "50")
	w.Header().Set("X-Output-Tokens", "100")
	w.Header().Set("X-Cost", "0.0025")
	w.Header().Set("X-Model", model)

	json.NewEncoder(w).Encode(response)
}

func (m *MockServer) handleCompletion(w http.ResponseWriter, r *http.Request) {
	m.handleChatCompletion(w, r)
}

func calculateMockCost(model string, inputTokens, outputTokens int) float64 {
	var inputCost, outputCost float64

	switch model {
	case "gpt-4":
		inputCost = 0.03
		outputCost = 0.06
	case "gpt-3.5-turbo":
		inputCost = 0.0015
		outputCost = 0.002
	default:
		inputCost = 0.001
		outputCost = 0.002
	}

	return (float64(inputTokens)/1000.0)*inputCost + (float64(outputTokens)/1000.0)*outputCost
} 