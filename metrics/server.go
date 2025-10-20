package metrics

import (
	"expvar"
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	EmptyLLMResponse           = expvar.NewInt("empty_llm_response")
	SuccessfulLLMGen           = expvar.NewInt("succesful_llm_gen")
	FailedLLMGen               = expvar.NewInt("failed_llm_gen")
	TwitchConnectionCount      = expvar.NewInt("twitch_connection_count")
	TwitchMessageRecievedCount = expvar.NewInt("twitch_message_recieved_count")
	TwitchMessageSentCount     = expvar.NewInt("twitch_message_sent_count")
	DiscordMessageRecieved     = expvar.NewInt("discord_message_recieved")
	DiscordMessageSent         = expvar.NewInt("discord_message_sent")
	WebSearchSuccessCount      = expvar.NewInt("web_search_success_count")
	WebSearchFailCount         = expvar.NewInt("web_search_fail_count")

	// KeepAlive metrics
	HealthCheckAttempts    = expvar.NewInt("health_check_attempts")
	HealthCheckSuccesses   = expvar.NewInt("health_check_successes")
	HealthCheckFailures    = expvar.NewInt("health_check_failures")
	AlertsSent             = expvar.NewInt("alerts_sent")
	ServiceRecoveries      = expvar.NewInt("service_recoveries")
)

type Server struct {
	*http.Server
}

func SetupServer() *Server {

	// pprof is setup by importing the net/http/pprof package
	server := &http.Server{
		Addr:         ":6060",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// setup expvar cache
	EmptyLLMResponse.Set(0)
	SuccessfulLLMGen.Set(0)
	FailedLLMGen.Set(0)
	TwitchConnectionCount.Set(0)
	TwitchMessageRecievedCount.Set(0)
	TwitchMessageSentCount.Set(0)
	WebSearchSuccessCount.Set(0)
	WebSearchFailCount.Set(0)
	HealthCheckAttempts.Set(0)
	HealthCheckSuccesses.Set(0)
	HealthCheckFailures.Set(0)
	AlertsSent.Set(0)
	ServiceRecoveries.Set(0)

	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewBuildInfoCollector(),
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		collectors.NewExpvarCollector(
			map[string]*prometheus.Desc{
				"twitch_connection_count":       prometheus.NewDesc("twitch_connection_count", "number of times twitch connection was established", nil, nil),
				"twitch_message_recieved_count": prometheus.NewDesc("twitch_message_recieved_count", "number of times twitch recieved a message", nil, nil),
				"twitch_message_sent_count":     prometheus.NewDesc("twitch_message_sent_count", "number of times twitch sent a message", nil, nil),
				"empty_llm_response":            prometheus.NewDesc("empty_llm_response", "number of times llm responded with and empty string ", nil, nil),
				"successfull_llm_gen":           prometheus.NewDesc("successfull_llm_gen", "number of times llm generated a valid response", nil, nil),
				"failed_llm_gen":                prometheus.NewDesc("failed_llm_gen", "number of times errors occured in llm generation", nil, nil),
				"web_search_success_count":      prometheus.NewDesc("web_search_success_count", "number of successful web searches", nil, nil),
				"web_search_fail_count":         prometheus.NewDesc("web_search_fail_count", "number of failed web searches", nil, nil),
				"health_check_attempts":         prometheus.NewDesc("health_check_attempts", "total number of health check attempts", nil, nil),
				"health_check_successes":        prometheus.NewDesc("health_check_successes", "number of successful health checks", nil, nil),
				"health_check_failures":         prometheus.NewDesc("health_check_failures", "number of failed health checks", nil, nil),
				"alerts_sent":                   prometheus.NewDesc("alerts_sent", "number of alerts sent to Discord", nil, nil),
				"service_recoveries":            prometheus.NewDesc("service_recoveries", "number of service recovery events", nil, nil),
			},
		))

	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	http.HandleFunc("/healthz", healthzHandler)
	return &Server{server}
}

// healthzHandler returns a simple health check response
func healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (s *Server) Run() {
	s.ListenAndServe()
}
