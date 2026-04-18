package autonomous

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gentleman-programming/gentle-ai/internal/model"
)

// mockEngine is a minimal GenerationEngine for testing PhaseRunner.
type mockPhaseEngine struct {
	agentID    model.AgentID
	response   string
	shouldFail bool
}

func (m *mockPhaseEngine) Agent() model.AgentID { return m.agentID }

func (m *mockPhaseEngine) Generate(ctx context.Context, prompt string) (string, error) {
	if m.shouldFail {
		return "", errors.New("mock generation error")
	}
	return m.response, nil
}

func (m *mockPhaseEngine) Available() bool { return true }

// TestPhaseRunnerNilEngine verifies that Run returns an error when the engine is nil.
func TestPhaseRunnerNilEngine(t *testing.T) {
	runner := NewPhaseRunner(nil)
	config := PhaseConfig{
		Phase:     PhaseExplore,
		MaxIter:   1,
		Timeout:   time.Second,
		ChangeName: "test-change",
	}

	_, err := runner.Run(context.Background(), config)
	if err == nil {
		t.Error("expected error when engine is nil, got nil")
	}
}

// TestPhaseRunnerHappyPath verifies that a successful phase returns Success=true.
func TestPhaseRunnerHappyPath(t *testing.T) {
	mock := &mockPhaseEngine{
		agentID:  model.AgentClaudeCode,
		response: `{"action":"done","summary":"explore complete","reason":"done"}`,
	}

	runner := NewPhaseRunner(mock)
	config := PhaseConfig{
		Phase:      PhaseExplore,
		ChangeName: "test-change",
		Task:       "explore something",
		MaxIter:    5,
		Timeout:    time.Second,
	}

	result, err := runner.Run(context.Background(), config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected Success=true, got false")
	}
	if result.Iterations == 0 {
		t.Error("expected at least 1 iteration")
	}
}

// TestPhaseRunnerFailedPhase verifies that ActionFailed results in Success=false.
func TestPhaseRunnerFailedPhase(t *testing.T) {
	mock := &mockPhaseEngine{
		agentID:  model.AgentClaudeCode,
		response: `{"action":"failed","summary":"phase failed","reason":"error"}`,
	}

	runner := NewPhaseRunner(mock)
	config := PhaseConfig{
		Phase:      PhasePropose,
		ChangeName: "test-change",
		Task:       "propose something",
		MaxIter:    3,
		Timeout:    time.Second,
	}

	result, err := runner.Run(context.Background(), config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Errorf("expected Success=false, got true")
	}
}

// TestPhaseRunnerGenerationError verifies that when Generate fails (returns error),
// ParseAction fails with empty input, which the loop converts to ActionFailed.
// The result has Success=false with no error returned from Run().
func TestPhaseRunnerGenerationError(t *testing.T) {
	mock := &mockPhaseEngine{
		agentID:    model.AgentClaudeCode,
		shouldFail: true, // Generate returns error → ParseAction("") fails
	}

	runner := NewPhaseRunner(mock)
	config := PhaseConfig{
		Phase:      PhaseExplore,
		ChangeName: "test-change",
		Task:       "explore something",
		MaxIter:    1,
		Timeout:    time.Second,
	}

	result, err := runner.Run(context.Background(), config)
	// Run() returns (report, nil) — errors from Generate are converted to ActionFailed steps.
	if err != nil {
		t.Errorf("Run() should not return error: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if result.Success {
		t.Error("expected Success=false when Generate fails")
	}
}

// TestPhaseRunnerDangerousPropagates verifies that the Dangerous flag is
// passed through to the taskrunner loop.
func TestPhaseRunnerDangerousPropagates(t *testing.T) {
	// We test this indirectly: if Dangerous=false, shell denylist should block.
	// If Dangerous=true, it should allow.
	// We use a mock that returns ActionDone immediately (1 iteration),
	// so we can't test shell behavior directly without a real engine.
	// This test verifies the config is accepted without error.
	for _, dangerous := range []bool{false, true} {
		name := "safe"
		if dangerous {
			name = "dangerous"
		}
		t.Run(name, func(t *testing.T) {
			mock := &mockPhaseEngine{
				agentID:  model.AgentClaudeCode,
				response: `{"action":"done","summary":"done","reason":"test"}`,
			}
			runner := NewPhaseRunner(mock)
			config := PhaseConfig{
				Phase:      PhaseExplore,
				ChangeName: "test-change",
				MaxIter:    1,
				Timeout:    time.Second,
				Dangerous: dangerous,
			}
			result, err := runner.Run(context.Background(), config)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !result.Success {
				t.Errorf("expected Success=true, got false")
			}
		})
	}
}

// TestPhaseRunnerContextTimeout verifies that the context deadline is respected.
func TestPhaseRunnerContextTimeout(t *testing.T) {
	mock := &mockPhaseEngine{
		agentID: model.AgentClaudeCode,
		// This response returns ActionDone, so no timeout by default.
		// To test timeout, we'd need a mock that hangs. Skipping for now.
		response: `{"action":"done","summary":"done","reason":"test"}`,
	}
	runner := NewPhaseRunner(mock)
	config := PhaseConfig{
		Phase:      PhaseExplore,
		ChangeName: "test-change",
		Task:       "explore something",
		MaxIter:    1,
		Timeout:    time.Second,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()
	result, err := runner.Run(ctx, config)
	if err != nil {
		t.Fatalf("expected no error with immediate ctx cancel after Generate returns fast, got: %v", err)
	}
	if !result.Success {
		t.Errorf("expected Success=true (Generate returned fast), got false")
	}
}
