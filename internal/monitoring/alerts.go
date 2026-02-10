// Package monitoring - alerts.go flags anomalies and errors.
//
// DESIGN: AlertManager logs notable events at appropriate levels:
//   - FlagHighLatency:       Warn when request exceeds threshold
//   - FlagCompressionFailure: Error when compression pipe fails
//   - FlagProviderError:     Warn on upstream 4xx/5xx responses
//   - FlagPanic:             Error on recovered panics
package monitoring

import "time"

// AlertManager flags anomalies and errors.
type AlertManager struct {
	logger               *Logger
	highLatencyThreshold time.Duration
}

// NewAlertManager creates a new alert manager.
func NewAlertManager(logger *Logger, cfg AlertConfig) *AlertManager {
	threshold := cfg.HighLatencyThreshold
	if threshold == 0 {
		threshold = 5 * time.Second
	}
	return &AlertManager{logger: logger, highLatencyThreshold: threshold}
}

// FlagHighLatency logs when request latency exceeds threshold.
func (am *AlertManager) FlagHighLatency(requestID string, latency time.Duration, provider, path string) {
	if latency < am.highLatencyThreshold {
		return
	}
	am.logger.Warn().
		Str("request_id", requestID).
		Dur("latency", latency).
		Str("provider", provider).
		Msg("high_latency")
}

// FlagCompressionFailure logs compression pipe failure.
func (am *AlertManager) FlagCompressionFailure(requestID, pipe, strategy string, err error) {
	am.logger.Error().
		Str("request_id", requestID).
		Str("pipe", pipe).
		Err(err).
		Msg("compression_failed")
}

// FlagProviderError logs upstream provider error.
func (am *AlertManager) FlagProviderError(requestID, provider string, statusCode int, errorMsg string) {
	am.logger.Warn().
		Str("request_id", requestID).
		Str("provider", provider).
		Int("status", statusCode).
		Msg("provider_error")
}

// FlagInvalidRequest logs invalid request.
func (am *AlertManager) FlagInvalidRequest(requestID, reason string, details map[string]interface{}) {
	am.logger.Debug().
		Str("request_id", requestID).
		Str("reason", reason).
		Msg("invalid_request")
}

// FlagPanic logs recovered panic.
func (am *AlertManager) FlagPanic(requestID string, panicValue interface{}, stack string) {
	am.logger.Error().
		Str("request_id", requestID).
		Interface("panic", panicValue).
		Msg("panic_recovered")
}

// FlagUpstreamTimeout logs upstream timeout.
func (am *AlertManager) FlagUpstreamTimeout(requestID, provider, targetURL string, timeout time.Duration) {
	am.logger.Error().
		Str("request_id", requestID).
		Str("provider", provider).
		Dur("timeout", timeout).
		Msg("upstream_timeout")
}
