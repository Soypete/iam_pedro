package metrics

import (
	"testing"
)

func TestExpvarMetricsExist(t *testing.T) {
	t.Run("EmptyLLMResponseCount is defined", func(t *testing.T) {
		EmptyLLMResponseCount.Add(0)
	})

	t.Run("SuccessfulLLMGenCount is defined", func(t *testing.T) {
		SuccessfulLLMGenCount.Add(0)
	})

	t.Run("FailedLLMGenCount is defined", func(t *testing.T) {
		FailedLLMGenCount.Add(0)
	})

	t.Run("TwitchConnectionCount is defined", func(t *testing.T) {
		TwitchConnectionCount.Add(0)
	})

	t.Run("TwitchMessageRecievedCount is defined", func(t *testing.T) {
		TwitchMessageRecievedCount.Add(0)
	})

	t.Run("WebSearchSuccessCount is defined", func(t *testing.T) {
		WebSearchSuccessCount.Add(0)
	})

	t.Run("WebSearchFailCount is defined", func(t *testing.T) {
		WebSearchFailCount.Add(0)
	})

	t.Run("FAQCheckCount is defined", func(t *testing.T) {
		FAQCheckCount.Add(0)
	})

	t.Run("FAQMatchCount is defined", func(t *testing.T) {
		FAQMatchCount.Add(0)
	})

	t.Run("ModActionTotal is defined", func(t *testing.T) {
		ModActionTotal.Add(0)
	})
}

func TestPrometheusMetricsExist(t *testing.T) {
	t.Run("ModerationActionsTotal is defined", func(t *testing.T) {
		ModerationActionsTotal.WithLabelValues("test", "true").Add(0)
	})

	t.Run("ModerationEvaluationsTotal is defined", func(t *testing.T) {
		ModerationEvaluationsTotal.Add(0)
	})

	t.Run("ModerationDecisionDuration is defined", func(t *testing.T) {
		ModerationDecisionDuration.Observe(0.0)
	})
}

func TestServerStruct(t *testing.T) {
	t.Run("Server has http.Server embedded", func(t *testing.T) {
		server := &Server{}
		_ = server.Server
	})
}
