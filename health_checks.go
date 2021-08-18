package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/bwmarrin/discordgo"
)

func healthMonitor(s *discordgo.Session, port uint) {
	http.HandleFunc("/healthy", func(w http.ResponseWriter, r *http.Request) {
		latency := s.HeartbeatLatency()
		degraded := 10 * time.Second

		if latency >= degraded {
			http.Error(w, fmt.Sprintf("discord latency=%d ms, expecting < %d ms\n", latency.Milliseconds(), degraded.Milliseconds()), 500)
			return
		}
		fmt.Fprintf(w, "OK\n")
	})

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
