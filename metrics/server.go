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
	// Expvar metrics (legacy)
	EmptyLLMResponseCount      = expvar.NewInt("empty_llm_response_count")
	SuccessfulLLMGenCount      = expvar.NewInt("successful_llm_gen_count")
	FailedLLMGenCount          = expvar.NewInt("failed_llm_gen_count")
	TwitchConnectionCount      = expvar.NewInt("twitch_connection_count")
	TwitchMessageRecievedCount = expvar.NewInt("twitch_message_recieved_count")
	TwitchMessageSentCount     = expvar.NewInt("twitch_message_sent_count")
	DiscordMessageRecieved     = expvar.NewInt("discord_message_recieved")
	DiscordMessageSent         = expvar.NewInt("discord_message_sent")
	WebSearchSuccessCount      = expvar.NewInt("web_search_success_count")
	WebSearchFailCount         = expvar.NewInt("web_search_fail_count")

	// Prometheus metrics with labels
	DiscordCommandTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "discord_command_total",
			Help: "Total number of Discord commands invoked by command type",
		},
		[]string{"command"},
	)

	DiscordCommandErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "discord_command_errors",
			Help: "Total number of Discord command errors by command type",
		},
		[]string{"command"},
	)

	DiscordCommandDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "discord_command_duration_seconds",
			Help:    "Duration of Discord command execution in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"command"},
	)

	DiscordStumpPedroGames = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "discord_stump_pedro_games_total",
			Help: "Total number of 20 questions games by status (started, won, lost)",
		},
		[]string{"status"},
	)
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
	EmptyLLMResponseCount.Set(0)
	SuccessfulLLMGenCount.Set(0)
	FailedLLMGenCount.Set(0)
	TwitchConnectionCount.Set(0)
	TwitchMessageRecievedCount.Set(0)
	TwitchMessageSentCount.Set(0)
	DiscordMessageRecieved.Set(0)
	DiscordMessageSent.Set(0)
	WebSearchSuccessCount.Set(0)
	WebSearchFailCount.Set(0)

	reg := prometheus.NewRegistry()
	reg.MustRegister(
		collectors.NewBuildInfoCollector(),
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		collectors.NewExpvarCollector(
			map[string]*prometheus.Desc{
				"twitch_connection_count":       prometheus.NewDesc("twitch_connection_count", "number of times twitch connection was established", nil, nil),
				"twitch_message_recieved_count": prometheus.NewDesc("twitch_message_recieved_count", "number of times twitch recieved a message", nil, nil),
				"twitch_message_sent_count":     prometheus.NewDesc("twitch_message_sent_count", "number of times twitch sent a message", nil, nil),
				"discord_message_recieved":      prometheus.NewDesc("discord_message_recieved", "number of times discord received a message", nil, nil),
				"discord_message_sent":          prometheus.NewDesc("discord_message_sent", "number of times discord sent a message", nil, nil),
				"empty_llm_response_count":      prometheus.NewDesc("empty_llm_response_count", "number of times llm responded with and empty string ", nil, nil),
				"successful_llm_gen_count":      prometheus.NewDesc("successful_llm_gen_count", "number of times llm generated a valid response", nil, nil),
				"failed_llm_gen_count":          prometheus.NewDesc("failed_llm_gen_count", "number of times errors occured in llm generation", nil, nil),
				"web_search_success_count":      prometheus.NewDesc("web_search_success_count", "number of successful web searches", nil, nil),
				"web_search_fail_count":         prometheus.NewDesc("web_search_fail_count", "number of failed web searches", nil, nil),
			},
		),
		// Register Discord command metrics with labels
		DiscordCommandTotal,
		DiscordCommandErrors,
		DiscordCommandDuration,
		DiscordStumpPedroGames,
	)

	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	http.HandleFunc("/healthz", healthzHandler)
	return &Server{server}
}

// healthzHandler returns a simple health check response
func healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func (s *Server) Run() {
	_ = s.ListenAndServe()
}
