// Package decoder provides direct integration with rtlamr for decoding smart meter messages.
package decoder

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bemasher/rtlamr/protocol"

	// Import protocol parsers (side-effect imports to register parsers).
	_ "github.com/bemasher/rtlamr/idm"
	_ "github.com/bemasher/rtlamr/netidm"
	_ "github.com/bemasher/rtlamr/r900"
	_ "github.com/bemasher/rtlamr/r900bcd"
	_ "github.com/bemasher/rtlamr/scm"
	_ "github.com/bemasher/rtlamr/scmplus"

	"rtlsdr2mqtt/internal/config"
	"rtlsdr2mqtt/internal/sdr"
)

const (
	// DefaultSymbolLength is the default symbol length for decoding.
	DefaultSymbolLength = 72

	// DefaultWatchdogTimeout is how long the decode loop may go without
	// receiving a sample block from the SDR before the decoder is restarted.
	// Sample blocks arrive every few milliseconds, so a silent device (USB
	// wedge, dead RF pipe) is flagged long before the health check file stales.
	DefaultWatchdogTimeout = 30 * time.Second
)

var (
	ErrDecoderNotStarted  = errors.New("decoder is not started")
	ErrDecoderStopped     = errors.New("decoder has been stopped")
	ErrDecoderTimeout     = errors.New("decoder timeout while reading messages")
	ErrShortRead          = errors.New("short read from SDR device")
	ErrSampleFlowStalled  = errors.New("sample flow from SDR stalled")
	ErrSampleStreamClosed = errors.New("sample stream from SDR closed unexpectedly")
	ErrInvalidBlockSize   = errors.New("invalid block size from decoder configuration")
)

// Decoder wraps the rtlamr decoder for direct integration.
type Decoder struct {
	sdr     sdr.SDR
	decoder protocol.Decoder
	fc      filterChain
	config  *config.Config
	logger  *slog.Logger

	watchdogTimeout time.Duration
	cancelFunc      context.CancelFunc
	doneChan        chan struct{}
	wg              sync.WaitGroup

	msgChan      chan *Message
	errChan      chan error
	sampleSignal chan struct{}
	isRunning    atomic.Bool
	mu           sync.Mutex // Only used for Start/Stop synchronization
}

// Message represents a decoded meter message.
type Message struct {
	Time        time.Time
	MeterID     uint32
	MeterType   uint8
	Consumption uint32
	Protocol    string
	Attributes  map[string]any
	Raw         protocol.Message
}

// NewDecoder creates a new decoder instance.
func NewDecoder(cfg *config.Config, logger *slog.Logger) *Decoder {
	if logger == nil {
		logger = slog.Default()
	}

	return &Decoder{
		config:          cfg,
		logger:          logger,
		watchdogTimeout: DefaultWatchdogTimeout,
		msgChan:         make(chan *Message, 100),
		errChan:         make(chan error, 10),
		sampleSignal:    make(chan struct{}, 1),
		doneChan:        make(chan struct{}),
	}
}

// Start connects to rtl_tcp and begins decoding.
func (d *Decoder) Start(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.isRunning.Load() {
		return nil
	}

	// Create internal context
	ctx, cancel := context.WithCancel(ctx)
	d.cancelFunc = cancel

	// Initialize the decoder
	d.decoder = protocol.NewDecoder()

	// Register protocols based on configuration
	protocols := d.getProtocols()
	for _, name := range protocols {
		p, err := protocol.NewParser(name, DefaultSymbolLength)
		if err != nil {
			return fmt.Errorf("failed to create parser for %s: %w", name, err)
		}
		d.decoder.RegisterProtocol(p)
	}

	// Allocate decoder buffers
	d.decoder.Allocate()

	// Build meter ID filter
	d.fc = filterChain{}
	meterIDs := d.getMeterIDs()
	if len(meterIDs) > 0 {
		d.fc.add(newMeterIDFilter(meterIDs))
	}

	// Create and open SDR device
	var device sdr.SDR
	if d.sdr != nil {
		// Pre-injected device (tests)
		device = d.sdr
	} else {
		var err error
		device, err = sdr.NewSDR(d.config, d.logger)
		if err != nil {
			return fmt.Errorf("failed to create SDR: %w", err)
		}
	}
	d.sdr = device

	// Open the device
	if err := d.sdr.Open(); err != nil {
		return fmt.Errorf("failed to open RTL-SDR device: %w", err)
	}

	// Configure SDR
	cfg := d.decoder.Cfg
	if err := d.sdr.SetCenterFreq(cfg.CenterFreq); err != nil {
		return fmt.Errorf("failed to set center frequency: %w", err)
	}
	// #nosec G115 - cfg.SampleRate is from protocol config, known safe values
	if err := d.sdr.SetSampleRate(uint32(cfg.SampleRate)); err != nil {
		return fmt.Errorf("failed to set sample rate: %w", err)
	}

	// Apply additional SDR configuration from config
	if err := sdr.ApplyConfiguration(d.sdr, d.config, d.logger); err != nil {
		return fmt.Errorf("failed to apply SDR configuration: %w", err)
	}

	// Reset buffer before starting reads
	if err := d.sdr.ResetBuffer(); err != nil {
		return fmt.Errorf("failed to reset buffer: %w", err)
	}

	// Each Decode call consumes exactly BlockSize2 bytes, so the sample
	// stream must deliver blocks of that size. librtlsdr's async buf_len is
	// in bytes of IQ data.
	if cfg.BlockSize2 <= 0 {
		return fmt.Errorf("%w: %d", ErrInvalidBlockSize, cfg.BlockSize2)
	}
	bufLen := uint32(cfg.BlockSize2) //nolint:gosec // BlockSize2 is a positive power-of-two byte count

	// Begin async sample delivery; blocks are dropped if the decode loop stalls.
	streamChan, err := d.sdr.StartStreaming(bufLen, 0)
	if err != nil {
		return fmt.Errorf("failed to start sample streaming: %w", err)
	}

	d.logger.Info("Decoder connected to RTL-SDR",
		"center_freq", cfg.CenterFreq,
		"sample_rate", cfg.SampleRate,
	)

	d.isRunning.Store(true)

	// Start the decode loop and watchdog
	d.wg.Add(2)
	go d.decodeLoop(ctx, streamChan)
	go d.watchdogLoop(ctx)

	return nil
}

// Stop stops the decoder and disconnects from rtl_tcp.
func (d *Decoder) Stop() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.isRunning.Load() {
		return nil
	}

	d.logger.Info("Stopping decoder...")

	// Cancel context to signal goroutines
	if d.cancelFunc != nil {
		d.cancelFunc()
	}
	close(d.doneChan)

	// Wait for goroutines to finish
	d.wg.Wait()

	// Stop the async sample reader and close the device
	if err := d.sdr.StopStreaming(); err != nil {
		d.logger.Error("Failed to stop sample streaming", "error", err)
	}
	if err := d.sdr.Close(); err != nil {
		d.logger.Error("Failed to close SDR device", "error", err)
	}

	d.isRunning.Store(false)

	// Recreate channels for potential restart
	d.msgChan = make(chan *Message, 100)
	d.errChan = make(chan error, 10)
	d.sampleSignal = make(chan struct{}, 1)
	d.doneChan = make(chan struct{})

	d.logger.Info("Decoder stopped")

	return nil
}

// IsRunning returns whether the decoder is running.
// This is lock-free using atomic operations.
func (d *Decoder) IsRunning() bool {
	return d.isRunning.Load()
}

// Channels returns the decoder's message, error, and done channels for direct access.
// This allows the caller to use a select statement with other channels.
func (d *Decoder) Channels() (<-chan *Message, <-chan error, <-chan struct{}) {
	return d.msgChan, d.errChan, d.doneChan
}

// ReadMessage reads the next decoded message with a timeout.
// Note: For better performance in loops, consider using Channels() directly.
func (d *Decoder) ReadMessage(timeout time.Duration) (*Message, error) {
	if !d.IsRunning() {
		return nil, ErrDecoderNotStarted
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case msg := <-d.msgChan:
		return msg, nil
	case err := <-d.errChan:
		return nil, err
	case <-timer.C:
		return nil, ErrDecoderTimeout
	case <-d.doneChan:
		return nil, ErrDecoderStopped
	}
}

// decodeLoop is the main decode loop that reads samples and decodes messages.
func (d *Decoder) decodeLoop(ctx context.Context, streamChan <-chan []byte) {
	defer d.wg.Done()

	// Track messages across blocks to deduplicate
	prev := make(map[protocol.Digest]bool)
	next := make(map[protocol.Digest]bool)

	for {
		select {
		case <-ctx.Done():
			return
		case block, ok := <-streamChan:
			if !ok {
				// Stream closed unexpectedly (device death); restart via error path.
				d.logger.Error("Sample stream closed unexpectedly")
				d.sendError(ErrSampleStreamClosed)
				return
			}

			// Notify the watchdog that samples are flowing.
			select {
			case d.sampleSignal <- struct{}{}:
			default:
			}

			// Decode reads exactly BlockSize2 input bytes; a short block
			// (flaky USB transfer) would index out of range, so drop it.
			if len(block) != d.decoder.Cfg.BlockSize2 {
				d.logger.Warn("Dropping sample block with unexpected size",
					"got", len(block), "want", d.decoder.Cfg.BlockSize2)
				continue
			}

			// Swap digest maps
			prev, next = next, prev

			// Clear next map
			for key := range next {
				delete(next, key)
			}

			// Decode messages from the block
			for msg := range d.decoder.Decode(block) {
				// Apply filters
				if !d.fc.match(msg) {
					continue
				}

				// Deduplicate messages spanning blocks
				digest := protocol.NewDigest(msg)
				next[digest] = true
				if prev[digest] {
					continue
				}

				// Convert to our message type
				decoded := d.convertMessage(msg)

				// Send message (non-blocking)
				select {
				case d.msgChan <- decoded:
				default:
					d.logger.Warn("Message channel full, dropping message",
						"meter_id", decoded.MeterID)
				}
			}
		}
	}
}

// watchdogLoop reports ErrSampleFlowStalled when the SDR stops delivering
// sample blocks. Healthy devices deliver a block every few milliseconds, so
// silence indicates a wedged USB pipe or dead reader. The error flows through
// errChan into the controller's existing decoder-restart path.
func (d *Decoder) watchdogLoop(ctx context.Context) {
	defer d.wg.Done()

	timer := time.NewTimer(d.watchdogTimeout)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-d.sampleSignal:
			// Sample block observed; restart the stall timer. The non-blocking
			// drain handles a value that fired concurrently with this branch.
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(d.watchdogTimeout)
		case <-timer.C:
			d.logger.Error("Sample flow stalled, initiating decoder restart",
				"timeout", d.watchdogTimeout)
			d.sendError(ErrSampleFlowStalled)
			return
		}
	}
}

// convertMessage converts a protocol.Message to our Message type.
func (d *Decoder) convertMessage(msg protocol.Message) *Message {
	decoded := &Message{
		Time:      time.Now(),
		MeterID:   msg.MeterID(),
		MeterType: msg.MeterType(),
		Protocol:  strings.ToLower(msg.MsgType()),
		Raw:       msg,
	}

	// Extract consumption and attributes based on message type
	decoded.Consumption, decoded.Attributes = extractMessageData(msg)

	return decoded
}

// sendError sends an error to the error channel (non-blocking).
func (d *Decoder) sendError(err error) {
	select {
	case d.errChan <- err:
	default:
	}
}

// getProtocols returns the list of protocols to decode based on configuration.
func (d *Decoder) getProtocols() []string {
	protocolSet := make(map[string]bool)
	for i := range d.config.Meters {
		proto := strings.ToLower(d.config.Meters[i].Protocol)
		protocolSet[proto] = true
	}

	protocols := make([]string, 0, len(protocolSet))
	for proto := range protocolSet {
		protocols = append(protocols, proto)
	}

	// Default to common protocols if none specified
	if len(protocols) == 0 {
		protocols = []string{"scm", "scm+", "idm", "r900"}
	}

	return protocols
}

// getMeterIDs returns the list of meter IDs to filter for.
func (d *Decoder) getMeterIDs() []uint32 {
	ids := make([]uint32, 0, len(d.config.Meters))
	for i := range d.config.Meters {
		// Parse meter ID string to uint32
		var id uint32
		if _, err := fmt.Sscanf(d.config.Meters[i].ID, "%d", &id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

// MeterIDString returns the meter ID as a string.
func (m *Message) MeterIDString() string {
	return strconv.FormatUint(uint64(m.MeterID), 10)
}

// ConsumptionInt64 returns the consumption as an int64.
func (m *Message) ConsumptionInt64() int64 {
	return int64(m.Consumption)
}
