package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type CrawlerDefaults struct {
	MaxDepth            int
	MaxPages            int
	TimeBudget          time.Duration
	MaxLinksPerPage     int
	GlobalConcurrency   int
	PerHostConcurrency  int
	UserAgent           string
	RespectRobots       bool
	RequestTimeout      time.Duration
	HeaderTimeout       time.Duration
	TLSHandshakeTimeout time.Duration
	IdleConnTimeout     time.Duration
	MaxBodyBytes        int64
	RobotsTTL           time.Duration
	RetryMax            int
	RetryBaseDelay      time.Duration
	CircuitTripCount    int
	CircuitResetTime    time.Duration
}

type Config struct {
	Port          int
	AllowedOrigin string
	DataRoot      string
	SearchBaseURL string
	SearchAPIKey  string
	Defaults      CrawlerDefaults
}

func Load() Config {
	loadEnvFiles(".env", "../.env")

	return Config{
		Port:          getInt("PORT", 8080),
		AllowedOrigin: getString("ALLOWED_ORIGIN", "*"),
		DataRoot:      getPath("DATA_ROOT", filepath.Join("..", "data", "runs")),
		SearchBaseURL: getString("SEARCH_BASE_URL", "https://api.search.brave.com/res/v1/web/search"),
		SearchAPIKey:  getString("BRAVE_SEARCH_API_KEY", ""),
		Defaults: CrawlerDefaults{
			MaxDepth:            getInt("DEFAULT_MAX_DEPTH", 2),
			MaxPages:            getInt("DEFAULT_MAX_PAGES", 50),
			TimeBudget:          getDuration("DEFAULT_TIME_BUDGET", 3*time.Minute),
			MaxLinksPerPage:     getInt("DEFAULT_MAX_LINKS_PER_PAGE", 25),
			GlobalConcurrency:   getInt("DEFAULT_GLOBAL_CONCURRENCY", 16),
			PerHostConcurrency:  getInt("DEFAULT_PER_HOST_CONCURRENCY", 4),
			UserAgent:           getString("DEFAULT_USER_AGENT", "ArachneCrawler/2.0"),
			RespectRobots:       getBool("DEFAULT_RESPECT_ROBOTS", true),
			RequestTimeout:      getDuration("DEFAULT_REQUEST_TIMEOUT", 15*time.Second),
			HeaderTimeout:       getDuration("DEFAULT_HEADER_TIMEOUT", 10*time.Second),
			TLSHandshakeTimeout: getDuration("DEFAULT_TLS_TIMEOUT", 8*time.Second),
			IdleConnTimeout:     getDuration("DEFAULT_IDLE_CONN_TIMEOUT", 90*time.Second),
			MaxBodyBytes:        getInt64("DEFAULT_MAX_BODY_BYTES", 2<<20),
			RobotsTTL:           getDuration("DEFAULT_ROBOTS_TTL", 24*time.Hour),
			RetryMax:            getInt("DEFAULT_RETRY_MAX", 2),
			RetryBaseDelay:      getDuration("DEFAULT_RETRY_BASE_DELAY", 300*time.Millisecond),
			CircuitTripCount:    getInt("DEFAULT_CIRCUIT_TRIP", 5),
			CircuitResetTime:    getDuration("DEFAULT_CIRCUIT_RESET", 30*time.Second),
		},
	}
}

func loadEnvFiles(paths ...string) {
	for _, path := range paths {
		loadEnvFile(path)
	}
}

func loadEnvFile(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		_ = os.Setenv(key, value)
	}
}

func getPath(key, def string) string {
	raw := getString(key, def)
	if filepath.IsAbs(raw) {
		return raw
	}
	return filepath.Clean(raw)
}

func getString(key, def string) string {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		return val
	}
	return def
}

func getInt(key string, def int) int {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		if n, err := strconv.Atoi(val); err == nil {
			return n
		}
	}
	return def
}

func getInt64(key string, def int64) int64 {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		if n, err := strconv.ParseInt(val, 10, 64); err == nil {
			return n
		}
	}
	return def
}

func getBool(key string, def bool) bool {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			return b
		}
	}
	return def
}

func getDuration(key string, def time.Duration) time.Duration {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return def
}
