package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/arsac/qb-proxies/internal/config"
	"github.com/arsac/qb-proxies/internal/proxy"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	addr := flag.String("addr", ":8080", "listen address")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	handler, err := proxy.NewHandler(cfg)
	if err != nil {
		log.Fatalf("failed to create handler: %v", err)
	}

	log.Printf("starting rss proxy on %s", *addr)
	if err := http.ListenAndServe(*addr, handler); err != nil {
		log.Fatalf("server error: %v", err)
		os.Exit(1)
	}
}
