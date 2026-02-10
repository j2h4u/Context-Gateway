// Package monitoring - telemetry.go records events to JSONL files.
//
// DESIGN: Tracker writes structured events as JSONL (one JSON object per line):
//   - RequestEvent:         Every request through the gateway
//   - ExpandEvent:          Each expand_context call
//   - CompressionComparison: Original vs compressed content (debug mode)
//
// Events are appended to files immediately after each event for real-time logging.
package monitoring

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/rs/zerolog/log"
)

// Tracker handles telemetry event recording to file and stdout.
type Tracker struct {
	config             TelemetryConfig
	requestLogPath     string
	compressionLogPath string
	requestCount       int
	compressionCount   int
	mu                 sync.Mutex
}

// NewTracker creates a new telemetry tracker.
func NewTracker(cfg TelemetryConfig) (*Tracker, error) {
	t := &Tracker{
		config: cfg,
	}

	if !cfg.Enabled {
		return t, nil
	}

	// Store paths and ensure directories exist, create empty files
	if cfg.LogPath != "" {
		if err := os.MkdirAll(filepath.Dir(cfg.LogPath), 0750); err != nil {
			return nil, err
		}
		t.requestLogPath = cfg.LogPath
		// Create empty file if it doesn't exist
		if _, err := os.Stat(cfg.LogPath); os.IsNotExist(err) {
			if f, err := os.Create(cfg.LogPath); err == nil {
				f.Close()
			}
		}
	}

	if cfg.CompressionLogPath != "" {
		if err := os.MkdirAll(filepath.Dir(cfg.CompressionLogPath), 0750); err != nil {
			return nil, err
		}
		t.compressionLogPath = cfg.CompressionLogPath
		// Create empty file if it doesn't exist
		if _, err := os.Stat(cfg.CompressionLogPath); os.IsNotExist(err) {
			if f, err := os.Create(cfg.CompressionLogPath); err == nil {
				f.Close()
			}
		}
	}

	return t, nil
}

// appendJSONL appends a single JSON object as a line to the file.
func appendJSONL(path string, event any) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	data = append(data, '\n')

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(data)
	return err
}

// RecordRequest records a request event.
func (t *Tracker) RecordRequest(event *RequestEvent) {
	if !t.config.Enabled {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	// Log summary to stdout if enabled
	if t.config.LogToStdout {
		reqID := event.RequestID
		if len(reqID) > 8 {
			reqID = reqID[:8]
		}
		log.Info().
			Str("request_id", reqID).
			Str("pipe", string(event.PipeType)).
			Int("tokens_saved", event.TokensSaved).
			Bool("success", event.Success).
			Msg("telemetry")
	}

	// Append to JSONL file
	if t.requestLogPath != "" {
		if err := appendJSONL(t.requestLogPath, event); err != nil {
			log.Error().Err(err).Str("path", t.requestLogPath).Msg("telemetry: failed to write request event")
		} else {
			t.requestCount++
		}
	}
}

// RecordExpand records an expand_context call.
func (t *Tracker) RecordExpand(event *ExpandEvent) {
	if !t.config.Enabled {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	// Append to JSONL file
	if t.requestLogPath != "" {
		if err := appendJSONL(t.requestLogPath, event); err != nil {
			log.Error().Err(err).Str("path", t.requestLogPath).Msg("telemetry: failed to write expand event")
		} else {
			t.requestCount++
		}
	}
}

// CompressionLogEnabled returns true if compression logging is enabled.
func (t *Tracker) CompressionLogEnabled() bool {
	return t.config.Enabled && t.compressionLogPath != ""
}

// LogCompressionComparison logs a compression comparison for debugging.
func (t *Tracker) LogCompressionComparison(comparison CompressionComparison) {
	if !t.CompressionLogEnabled() {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	// Append to JSONL file
	if err := appendJSONL(t.compressionLogPath, comparison); err != nil {
		log.Error().Err(err).Str("path", t.compressionLogPath).Msg("telemetry: failed to write compression event")
	} else {
		t.compressionCount++
	}
}

// Close is kept for interface compatibility.
func (t *Tracker) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.requestLogPath != "" && t.requestCount > 0 {
		log.Info().
			Str("path", t.requestLogPath).
			Int("events", t.requestCount).
			Msg("telemetry: session complete")
	}

	return nil
}
