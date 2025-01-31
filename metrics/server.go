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
			},
		))

	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	return &Server{server}
}

func (s *Server) Run() {
	s.ListenAndServe()
}
