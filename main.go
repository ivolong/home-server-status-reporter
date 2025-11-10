package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
)

type HealthCheck struct {
	Name        string        `json:"name"`
	Description string        `json:"description`
	Icon        template.HTML `json:"icon"`
	Endpoint    string        `json:"endpoint"`
	StatusCode  int           `json:"status_code"`
	Healthy     bool          `json:"healthy"`
}

type SystemStats struct {
	CPU           []float64
	MemoryUsed    uint64
	MemoryTotal   uint64
	MemoryPercent float64
	DiskUsed      uint64
	DiskTotal     uint64
	DiskPercent   float64
	LastUpdated   time.Time
}

type Config struct {
	Site                   string        `json:"site"`
	Port                   int           `json:"port"`
	RefreshIntervalSeconds int           `json:"refresh_interval_seconds"`
	HealthChecks           []HealthCheck `json:"healthchecks"`
}

type TemplateData struct {
	Config   Config
	Stats    SystemStats
	Services []HealthCheck
	Uptime   time.Duration
	Updated  time.Duration
}

func formatBytes(b uint64) string {
	if b == 0 {
		return "0 B"
	}

	const unit = 1024
	f := float64(b)
	div := 0
	units := []string{"B", "KB", "MB", "GB", "TB", "PB"}
	for f >= unit && div < len(units)-1 {
		f /= unit
		div++
	}

	return fmt.Sprintf("%.2f %s", f, units[div])
}

func formatPercent(p float64) string {
	return fmt.Sprintf("%.2f%%", p)
}

func collectStats() {
	for {
		cpuPercent, err := cpu.Percent(0, false)
		if err != nil || len(cpuPercent) == 0 {
			log.Printf("Error getting CPU percent: %v", err)
		}

		memInfo, err := mem.VirtualMemory()
		if err != nil {
			log.Printf("Error getting memory info: %v", err)
		}

		diskInfo, err := disk.Usage("/")
		if err != nil {
			log.Printf("Error getting disk info: %v", err)
		}

		newHealthchecks := healthchecks
		for i, healthcheck := range healthchecks {
			newHealthchecks[i].Healthy = true

			response, err := http.Get(healthcheck.Endpoint)
			if err != nil {
				log.Printf("Error checking health: %v", err)
				newHealthchecks[i].Healthy = false
				continue
			}
			defer response.Body.Close()

			if response.StatusCode != healthcheck.StatusCode {
				newHealthchecks[i].Healthy = false
			}
		}

		reportMutex.Lock()
		healthchecks = newHealthchecks
		stats = SystemStats{
			CPU:           cpuPercent,
			MemoryUsed:    memInfo.Used,
			MemoryTotal:   memInfo.Total,
			MemoryPercent: memInfo.UsedPercent,
			DiskUsed:      diskInfo.Used,
			DiskTotal:     diskInfo.Total,
			DiskPercent:   diskInfo.UsedPercent,
			LastUpdated:   time.Now(),
		}
		reportMutex.Unlock()

		time.Sleep(time.Duration(config.RefreshIntervalSeconds) * time.Second)
	}
}

var (
	config       Config
	healthchecks []HealthCheck
	stats        SystemStats
	reportMutex  sync.RWMutex
)

func main() {
	configFile, err := ioutil.ReadFile("config.json")
	if err != nil {
		log.Fatalf("Failed to load config.json: %v", err)
	}

	err = json.Unmarshal(configFile, &config)
	if err != nil {
		log.Fatalf("Failed to parse config.json: %v", err)
	}

	healthchecks = config.HealthChecks

	funcs := template.FuncMap{
		"FormatPercent": formatPercent,
		"FormatBytes":   formatBytes,
	}
	tmpl, err := template.New("template.html").Funcs(funcs).ParseFiles("template.html")
	if err != nil {
		log.Fatalf("Error parsing template: %v", err)
	}

	startTime := time.Now()

	go collectStats()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		reportMutex.RLock()
		defer reportMutex.RUnlock()

		templateData := TemplateData{
			Config:   config,
			Stats:    stats,
			Services: healthchecks,
			Uptime:   time.Since(startTime).Round(time.Second),
			Updated:  time.Since(stats.LastUpdated).Round(time.Second),
		}

		if err := tmpl.Execute(w, templateData); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	port := fmt.Sprintf(":%d", config.Port)
	log.Println("Serving system stats on http://localhost" + port)
	log.Fatal(http.ListenAndServe(port, nil))
}
