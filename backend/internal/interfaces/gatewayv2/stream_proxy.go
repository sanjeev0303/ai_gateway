// Package gatewayv2 implements the latency-sensitive data-plane gateway.
package gatewayv2

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptrace"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

var (
	ErrClientDisconnected = errors.New("client disconnected")
	ErrUpstreamTimeout    = errors.New("upstream timeout")
	ErrUpstreamError      = errors.New("upstream error")
	ErrStreamTruncated    = errors.New("stream truncated")
)

// StreamProxyImpl implements high-performance streaming proxy with HTTP/2 support.
type StreamProxyImpl struct {
	client          *http.Client
	readBufferSize  int
	writeBufferSize int
	logger          *slog.Logger
	mu              sync.RWMutex
	activeStreams   map[uuid.UUID]*activeStream
	streamCount     int64
	maxStreams      int64
}

type activeStream struct {
	id          uuid.UUID
	startTime   time.Time
	clientConn  io.Closer
	upstreamReq *http.Request
	cancel      context.CancelFunc
}

const (
	defaultReadBufferSize  = 32 * 1024
	defaultWriteBufferSize = 32 * 1024
	defaultMaxStreams      = 10000
)

// NewStreamProxy creates a new stream proxy.
func NewStreamProxy(client *http.Client, logger *slog.Logger) *StreamProxyImpl {
	if client == nil {
		client = defaultHTTPClient()
	}
	if logger == nil {
		logger = slog.Default()
	}

	return &StreamProxyImpl{
		client:          client,
		readBufferSize:  defaultReadBufferSize,
		writeBufferSize: defaultWriteBufferSize,
		logger:          logger,
		activeStreams:   make(map[uuid.UUID]*activeStream),
		maxStreams:      defaultMaxStreams,
	}
}

// defaultHTTPClient returns a client optimized for streaming proxies.
func defaultHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			// HTTP/2 settings
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          1000,
			MaxIdleConnsPerHost:   100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			// Disable compression for byte-exact streaming
			DisableCompression: true,
			// Response header timeout for upstream
			ResponseHeaderTimeout: 30 * time.Second,
		},
		// No overall timeout - streaming requests may run for minutes
		// Request context handles cancellation
		Timeout: 0,
	}
}

// Forward forwards the request to upstream and streams the response byte-exact.
func (s *StreamProxyImpl) Forward(ctx context.Context, w http.ResponseWriter, r *http.Request, upstreamURL string, upstreamHeaders http.Header) (*StreamUsage, error) {
	// Check stream limit
	if atomic.LoadInt64(&s.streamCount) >= s.maxStreams {
		s.logger.Warn("stream limit exceeded", "count", s.streamCount, "max", s.maxStreams)
		return nil, errors.New("gateway: stream limit exceeded")
	}

	// Create stream context with cancellation on client disconnect
	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Track active stream
	streamID := uuid.New()
	stream := &activeStream{
		id:        streamID,
		startTime: time.Now(),
		cancel:    cancel,
	}

	// Register stream
	s.mu.Lock()
	if len(s.activeStreams) >= int(s.maxStreams) {
		s.mu.Unlock()
		return nil, errors.New("gateway: stream limit exceeded")
	}
	s.activeStreams[streamID] = stream
	atomic.AddInt64(&s.streamCount, 1)
	s.mu.Unlock()

	// Ensure cleanup
	defer func() {
		s.mu.Lock()
		delete(s.activeStreams, streamID)
		atomic.AddInt64(&s.streamCount, -1)
		s.mu.Unlock()
	}()

	// Client-disconnect cancellation needs no extra watcher: streamCtx derives
	// from the request context, which net/http already cancels when the client
	// goes away. The previous http.CloseNotifier assertion panicked outright on
	// any writer that is not HTTP/1, including wrapped and test writers.
	clientGone := ctx.Done()
	go func() {
		select {
		case <-clientGone:
			s.logger.Debug("client disconnected, cancelling upstream", "stream_id", streamID)
		case <-streamCtx.Done():
		}
	}()

	// Build upstream request
	upstreamReq, err := s.buildUpstreamRequest(streamCtx, r, upstreamURL, upstreamHeaders)
	if err != nil {
		return nil, err
	}
	stream.upstreamReq = upstreamReq

	// Add tracing for latency measurement
	trace := &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) {
			s.logger.Debug("upstream connection established", "stream_id", streamID, "reused", info.Reused)
		},
		GotFirstResponseByte: func() {
			s.logger.Debug("upstream first byte", "stream_id", streamID, "elapsed_ms", time.Since(stream.startTime).Milliseconds())
		},
	}
	upstreamReq = upstreamReq.WithContext(httptrace.WithClientTrace(streamCtx, trace))

	// Execute upstream request
	resp, err := s.client.Do(upstreamReq)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, ErrClientDisconnected
		}
		s.logger.Error("upstream request failed", "stream_id", streamID, "error", err)
		return nil, ErrUpstreamError
	}
	defer resp.Body.Close()

	// Check for upstream errors
	if resp.StatusCode >= 400 {
		s.logger.Warn("upstream error response", "stream_id", streamID, "status", resp.StatusCode)
		// Still stream the error response to client for provider-native errors
	}

	// Copy response headers (excluding hop-by-hop)
	copyResponseHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)

	// Stream response body with flushing, observing provider-reported usage as
	// the bytes pass. The observer only reads; the bytes written to w are
	// exactly the bytes the upstream sent.
	obs := newUsageObserver(resp.Header.Get("Content-Type"))
	usage, err := s.streamResponse(streamCtx, w, &observingReader{src: resp.Body, obs: obs}, obs)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return usage, ErrClientDisconnected
		}
		if strings.Contains(err.Error(), "broken pipe") || strings.Contains(err.Error(), "connection reset") {
			return usage, ErrClientDisconnected
		}
		s.logger.Error("stream copy failed", "stream_id", streamID, "error", err)
		return usage, ErrStreamTruncated
	}

	return usage, nil
}

// buildUpstreamRequest constructs the upstream HTTP request.
func (s *StreamProxyImpl) buildUpstreamRequest(ctx context.Context, r *http.Request, upstreamURL string, upstreamHeaders http.Header) (*http.Request, error) {
	// Preserve the exact path and query
	targetURL := upstreamURL + strings.TrimPrefix(r.URL.RequestURI(), "/")

	outReq, err := http.NewRequestWithContext(ctx, r.Method, targetURL, r.Body)
	if err != nil {
		return nil, err
	}

	// Preserve Content-Length
	if r.ContentLength >= 0 {
		outReq.ContentLength = r.ContentLength
	}

	// Copy forward headers (excluding hop-by-hop and auth)
	copyForwardHeaders(outReq.Header, r.Header)

	// Inject upstream headers
	for k, v := range upstreamHeaders {
		for _, vv := range v {
			outReq.Header.Add(k, vv)
		}
	}

	return outReq, nil
}

// streamResponse copies the upstream response to the client with flushing,
// reporting the usage the observer saw. Usage is reported on every exit path:
// a client that disconnects after the final usage frame was still billed.
func (s *StreamProxyImpl) streamResponse(ctx context.Context, w http.ResponseWriter, src io.Reader, obs *usageObserver) (*StreamUsage, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		s.logger.Warn("ResponseWriter does not support flushing")
	}

	buf := make([]byte, s.readBufferSize)
	for {
		select {
		case <-ctx.Done():
			return obs.finish(), ErrClientDisconnected
		default:
		}

		n, err := src.Read(buf)
		if n > 0 {
			if _, werr := w.Write(buf[:n]); werr != nil {
				if strings.Contains(werr.Error(), "broken pipe") || strings.Contains(werr.Error(), "connection reset") {
					return obs.finish(), ErrClientDisconnected
				}
				return obs.finish(), werr
			}
			if flusher != nil {
				flusher.Flush()
			}
		}

		if err != nil {
			if err == io.EOF {
				return obs.finish(), nil
			}
			return obs.finish(), err
		}
	}
}

// Close gracefully shuts down the proxy, waiting for active streams to complete.
func (s *StreamProxyImpl) Close(gracePeriod time.Duration) error {
	s.logger.Info("shutting down stream proxy", "active_streams", len(s.activeStreams))

	// Cancel all active streams
	s.mu.Lock()
	for _, stream := range s.activeStreams {
		stream.cancel()
	}
	s.mu.Unlock()

	// Wait for streams to drain
	deadline := time.Now().Add(gracePeriod)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for time.Now().Before(deadline) {
		s.mu.RLock()
		count := len(s.activeStreams)
		s.mu.RUnlock()

		if count == 0 {
			s.logger.Info("all streams drained")
			return nil
		}
		<-ticker.C
	}

	// Force cancel remaining
	s.mu.Lock()
	for _, stream := range s.activeStreams {
		stream.cancel()
	}
	s.mu.Unlock()

	s.logger.Warn("grace period exceeded, forced shutdown", "remaining", len(s.activeStreams))
	return nil
}

// ActiveStreamCount returns the number of currently active streams.
func (s *StreamProxyImpl) ActiveStreamCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.activeStreams)
}

// SetMaxStreams sets the maximum concurrent streams.
func (s *StreamProxyImpl) SetMaxStreams(max int64) {
	atomic.StoreInt64(&s.maxStreams, max)
}

// Stats returns proxy statistics.
func (s *StreamProxyImpl) Stats() (activeStreams int64, maxStreams int64) {
	return atomic.LoadInt64(&s.streamCount), s.maxStreams
}

// copyResponseHeaders copies end-to-end headers from upstream to client.
func copyResponseHeaders(dst, src http.Header) {
	hopByHop := map[string]bool{
		"connection":          true,
		"keep-alive":          true,
		"proxy-authenticate":  true,
		"proxy-authorization": true,
		"te":                  true,
		"trailer":             true,
		"transfer-encoding":   true,
		"upgrade":             true,
		"content-length":      true, // Let Go set this
	}

	for key, values := range src {
		if hopByHop[strings.ToLower(key)] {
			continue
		}
		for _, v := range values {
			dst.Add(key, v)
		}
	}
}

// copyForwardHeaders copies end-to-end headers from client to upstream.
func copyForwardHeaders(dst, src http.Header) {
	hopByHop := map[string]bool{
		"connection":          true,
		"keep-alive":          true,
		"proxy-authenticate":  true,
		"proxy-authorization": true,
		"te":                  true,
		"trailer":             true,
		"transfer-encoding":   true,
		"upgrade":             true,
		"host":                true,
		"content-length":      true,
		"authorization":       true,
		"x-api-key":           true,
	}

	for key, values := range src {
		if hopByHop[strings.ToLower(key)] {
			continue
		}
		for _, v := range values {
			dst.Add(key, v)
		}
	}
}
