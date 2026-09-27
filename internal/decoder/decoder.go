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
	sdr            sdr.SDR
	acquirer       sdr.DeviceSource
	deviceIdentity string
	decoder        protocol.Decoder
	fc             filterChain
	config         *config.Config
	logger         *slog.Logger

	watchdogTimeout time.Duration
	cancelFunc      context.CancelFunc
	doneChan        chan struct{}
	wg              sync.WaitGroup

	msgChan   chan *Message
	errChan   chan error
	isRunning atomic.Bool
	mu        sync.Mutex // Only used for Start/Stop synchronization

	// Recovery support. All fields are plain data: they are only touched by
	// the supervisor goroutine, or set before Start.
	onDeviceAcquired func()
	sleeper          func(ctx context.Context, d time.Duration) error
	now              func() time.Time
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
		acquirer:        sdr.NewDeviceSource(cfg, logger),
		logger:          logger,
		watchdogTimeout: DefaultWatchdogTimeout,
		msgChan:         make(chan *Message, 100),
		errChan:         make(chan error, 10),
		doneChan:        make(chan struct{}),
		sleeper:         sleepContext,
		now:             time.Now,
	}
}

// SetDeviceSource replaces the device acquisition source. Tests substitute a
// fake that scripts device appearance, disappearance, and open failures.
func (d *Decoder) SetDeviceSource(source sdr.DeviceSource) {
	d.acquirer = source
}

// SetOnDeviceAcquired registers a callback invoked once per successful device
// (re-)acquisition, after the device is fully configured and streaming. The
// controller uses it to refresh the health check heartbeat. It is never
// invoked for unsuccessful acquisition attempts.
func (d *Decoder) SetOnDeviceAcquired(fn func()) {
	d.onDeviceAcquired = fn
}

// SetSleeper replaces the recovery backoff wait. Tests substitute an instant
// or scripted sleeper so backoff tests do not sleep in real time.
func (d *Decoder) SetSleeper(sleeper func(ctx context.Context, d time.Duration) error) {
	d.sleeper = sleeper
}

// DeviceIdentity returns a short identity string for the currently attached
// device, for logs. It is empty before the first successful acquisition.
func (d *Decoder) DeviceIdentity() string {
	return d.deviceIdentity
}

// sleepContext waits for d or until ctx is canceled.
func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return fmt.Errorf("recovery backoff wait canceled: %w", ctx.Err())
	case <-timer.C:
		return nil
	}
}

// recoveryBackoffDelay returns the wait before recovery attempt n (1-based).
// The ladder is 1s, 2s, 5s, then the configured steady-state poll interval,
// which also caps the ladder: each step never exceeds cap.
func recoveryBackoffDelay(attempt int, maxDelay time.Duration) time.Duration {
	var wait time.Duration
	switch attempt {
	case 1:
		wait = time.Second
	case 2:
		wait = 2 * time.Second
	case 3:
		wait = 5 * time.Second
	default:
		wait = 10 * time.Second
	}
	return min(wait, maxDelay)
}

// Start connects to the RTL-SDR and begins decoding.
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

	// Acquire the device through fresh USB enumeration and fully configure it.
	// Both startup and re-enumeration recovery funnel through openDevice, so
	// configuration can never drift between the two paths.
	if d.acquirer == nil {
		d.acquirer = sdr.NewDeviceSource(d.config, d.logger)
	}
	device, stream, identity, err := d.openDevice()
	if err != nil {
		return err
	}
	d.sdr = device
	d.deviceIdentity = identity
	// Initial acquisition: the health heartbeat fires immediately so the
	// liveness probe reflects startup progress, not just meter traffic.
	d.notifyAcquired(true, 1, false)

	d.logger.Info("Decoder connected to RTL-SDR",
		"device", identity,
		"center_freq", d.decoder.Cfg.CenterFreq,
		"sample_rate", d.decoder.Cfg.SampleRate,
	)

	d.isRunning.Store(true)

	// The supervisor owns the pipeline lifecycle: decode sessions plus
	// device-loss recovery.
	d.wg.Add(1)
	go d.runLoop(ctx, stream)

	return nil
}

// Stop stops the decoder and releases the SDR device, including any device
// abandoned mid-recovery: after Stop the decoder holds no device handle.
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
	if d.sdr != nil {
		if err := d.sdr.StopStreaming(); err != nil {
			d.logger.Error("Failed to stop sample streaming", "error", err)
		}
		if err := d.sdr.Close(); err != nil {
			d.logger.Error("Failed to close SDR device", "error", err)
		}
		d.sdr = nil
	}
	d.deviceIdentity = ""

	d.isRunning.Store(false)

	// Recreate channels for potential restart
	d.msgChan = make(chan *Message, 100)
	d.errChan = make(chan error, 10)
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

// openDevice acquires an RTL-SDR through fresh USB enumeration, applies the
// full startup configuration, and starts async sample delivery. Any failure
// releases the partial handle before returning so no stale or half-open
// handle survives into the next attempt. Startup and re-enumeration recovery
// share this path, so configuration can never drift between them.
func (d *Decoder) openDevice() (sdr.SDR, sdr.Stream, string, error) {
	device, identity, err := d.acquirer.Acquire()
	if err != nil {
		return nil, sdr.Stream{}, "", fmt.Errorf("failed to acquire RTL-SDR device: %w", err)
	}

	if cfgErr := d.configureDevice(device); cfgErr != nil {
		_ = device.Close()
		return nil, sdr.Stream{}, "", fmt.Errorf("failed to configure RTL-SDR device: %w", cfgErr)
	}

	stream, err := d.startStream(device)
	if err != nil {
		_ = device.Close()
		return nil, sdr.Stream{}, "", err
	}

	return device, stream, identity, nil
}

// configureDevice applies the exact startup SDR configuration to a freshly
// acquired device: center frequency, sample rate, gain/AGC/PPM settings, and
// a buffer reset.
func (d *Decoder) configureDevice(device sdr.SDR) error {
	cfg := d.decoder.Cfg
	if err := device.SetCenterFreq(cfg.CenterFreq); err != nil {
		return fmt.Errorf("failed to set center frequency: %w", err)
	}
	// #nosec G115 - cfg.SampleRate is from protocol config, known safe values
	if err := device.SetSampleRate(uint32(cfg.SampleRate)); err != nil {
		return fmt.Errorf("failed to set sample rate: %w", err)
	}

	// Apply additional SDR configuration from config
	if err := sdr.ApplyConfiguration(device, d.config, d.logger); err != nil {
		return fmt.Errorf("failed to apply SDR configuration: %w", err)
	}

	// Reset buffer before starting reads
	if err := device.ResetBuffer(); err != nil {
		return fmt.Errorf("failed to reset buffer: %w", err)
	}

	return nil
}

// notifyAcquired invokes the on-acquire callback (health heartbeat) when the
// acquisition represents genuine progress: the initial startup acquisition,
// a recovery that overcame absence (more than one attempt), or a recovery
// after a session that actually streamed samples. A present-but-dead device
// that re-opens instantly with zero sample flow gets no touch, so the
// liveness probe still trips when no decode progress is possible.
func (d *Decoder) notifyAcquired(first bool, attempts int, prevProgress bool) {
	if d.onDeviceAcquired == nil {
		return
	}
	if first || attempts > 1 || prevProgress {
		d.onDeviceAcquired()
	}
}

// sessionOutcome reports how a decode session ended. A nil err means the
// session ended because ctx was canceled (Stop); lost marks loss-class
// endings that require USB re-enumeration, and progressed records whether
// the session delivered at least one sample block.
type sessionOutcome struct {
	err        error
	lost       bool
	progressed bool
}

// runLoop supervises the decode pipeline: it runs decode sessions and, on
// device loss, recovers the device in-process before resuming decoding. It
// exits only when ctx is canceled (Stop). A non-loss stream error keeps the
// existing semantics: it is forwarded on errChan and the loop ends, leaving
// restart decisions to the controller.
func (d *Decoder) runLoop(ctx context.Context, stream sdr.Stream) {
	defer d.wg.Done()

	// Whether a previous session in this run delivered at least one sample
	// block. A re-acquired device only refreshes the health heartbeat when
	// it represents genuine recovery (absence overcome or prior streaming),
	// so a present-but-dead device cannot hold the probe fresh forever.
	prevProgress := false

	for {
		outcome := d.decodeSession(ctx, stream)
		if outcome.err == nil {
			return
		}
		prevProgress = prevProgress || outcome.progressed

		if !outcome.lost {
			d.logger.Error("Sample stream failed with non-loss error", "error", outcome.err)
			d.sendError(outcome.err)
			return
		}

		newStream, ok := d.recoverDevice(ctx, outcome.err, prevProgress)
		if !ok {
			return
		}
		stream = newStream
		prevProgress = false
	}
}

// decodeSession runs one decode session over an open stream until the stream
// ends, the sample-flow watchdog stalls, or ctx is canceled. It reports the
// terminal error, whether it is loss-class (device must be re-enumerated)
// or not (existing semantics apply), and whether the session delivered at
// least one sample block. A nil error means ctx was canceled.
func (d *Decoder) decodeSession(ctx context.Context, stream sdr.Stream) sessionOutcome {
	// Track messages across blocks to deduplicate
	prev := make(map[protocol.Digest]bool)
	next := make(map[protocol.Digest]bool)

	stallTimer := time.NewTimer(d.watchdogTimeout)
	defer stallTimer.Stop()

	kickWatchdog := func() {
		if !stallTimer.Stop() {
			select {
			case <-stallTimer.C:
			default:
			}
		}
		stallTimer.Reset(d.watchdogTimeout)
	}

	progressed := false

	for {
		select {
		case <-ctx.Done():
			return sessionOutcome{progressed: progressed}
		case <-stallTimer.C:
			// Sample flow stalled: the device stopped delivering blocks.
			// Silence from a streaming RTL-SDR means a wedged USB pipe or a
			// dead dongle, so this is loss-class.
			d.logger.Error("Sample flow stalled, entering device recovery",
				"timeout", d.watchdogTimeout)
			return sessionOutcome{err: fmt.Errorf("%w: %w", sdr.ErrDeviceLost, ErrSampleFlowStalled), lost: true, progressed: progressed}
		case streamErr, ok := <-stream.Errors:
			if !ok {
				// Errors closed without a value only happens when the stream
				// ended without a classified cause; treat it as an
				// unexpected closure of a dead pipe.
				return sessionOutcome{err: fmt.Errorf("%w: %w", sdr.ErrDeviceLost, ErrSampleStreamClosed), lost: true, progressed: progressed}
			}
			if sdr.IsDeviceLost(streamErr) {
				return sessionOutcome{err: streamErr, lost: true, progressed: progressed}
			}
			return sessionOutcome{err: streamErr, progressed: progressed}
		case block, ok := <-stream.Samples:
			if !ok {
				// The sample channel closed. When the device reports a
				// terminal error it is already queued on Errors, so recover
				// the classified cause here before falling back to a plain
				// unexpected closure. Note the ok check: receiving from a
				// closed, drained Errors channel yields a nil error, which
				// must not masquerade as a clean shutdown.
				select {
				case streamErr, ok := <-stream.Errors:
					if ok {
						if sdr.IsDeviceLost(streamErr) {
							return sessionOutcome{err: streamErr, lost: true, progressed: progressed}
						}
						return sessionOutcome{err: streamErr, progressed: progressed}
					}
				default:
				}
				d.logger.Error("Sample stream closed unexpectedly")
				return sessionOutcome{err: fmt.Errorf("%w: %w", sdr.ErrDeviceLost, ErrSampleStreamClosed), lost: true, progressed: progressed}
			}

			progressed = true
			kickWatchdog()

			prev, next = d.decodeBlock(block, prev, next)
		}
	}
}

// decodeBlock runs one sample block through the decoder and forwards new
// meter messages, returning the swapped digest maps that deduplicate
// messages spanning blocks.
func (d *Decoder) decodeBlock(
	block []byte, prev, next map[protocol.Digest]bool,
) (map[protocol.Digest]bool, map[protocol.Digest]bool) {
	// Decode reads exactly BlockSize2 input bytes; a short block
	// (flaky USB transfer) would index out of range, so drop it.
	if len(block) != d.decoder.Cfg.BlockSize2 {
		d.logger.Warn("Dropping sample block with unexpected size",
			"got", len(block), "want", d.decoder.Cfg.BlockSize2)
		return prev, next
	}

	// Swap digest maps
	prev, next = next, prev

	// Clear next map
	for key := range next {
		delete(next, key)
	}

	// Decode messages from the block
	for msg := range d.decoder.Decode(block) {
		d.handleDecodedMessage(msg, prev, next)
	}

	return prev, next
}

// handleDecodedMessage applies the meter filter and cross-block dedup, then
// forwards new messages. It reports whether the message was forwarded.
func (d *Decoder) handleDecodedMessage(msg protocol.Message, prev, next map[protocol.Digest]bool) bool {
	// Apply filters
	if !d.fc.match(msg) {
		return false
	}

	// Deduplicate messages spanning blocks
	digest := protocol.NewDigest(msg)
	next[digest] = true
	if prev[digest] {
		return false
	}

	// Convert to our message type
	decoded := d.convertMessage(msg)

	// Send message (non-blocking)
	select {
	case d.msgChan <- decoded:
		return true
	default:
		d.logger.Warn("Message channel full, dropping message",
			"meter_id", decoded.MeterID)
		return false
	}
}

// recoverDevice closes the stale device handle, then polls for the RTL-SDR
// to re-enumerate with capped backoff until a fresh device is acquired,
// configured, and streaming. The old handle is always released before any
// reopen attempt, so a claimed interface cannot block the new open with
// "Device or resource busy".
//
// It returns the new stream, or false when ctx was canceled (shutdown); in
// that case no device is left open.
func (d *Decoder) recoverDevice(ctx context.Context, cause error, prevProgress bool) (sdr.Stream, bool) {
	startedAt := d.now()
	d.logger.Warn("RTL-SDR lost; entering re-enumeration recovery", "error", cause)

	// Release the stale handle before any reopen attempt.
	if d.sdr != nil {
		if err := d.sdr.Close(); err != nil {
			d.logger.Warn("Failed to close lost RTL-SDR device; reopen may stay blocked until re-enumeration",
				"error", err)
		}
		d.sdr = nil
	}
	d.deviceIdentity = ""

	capDelay := d.config.SDR.DeviceRetryInterval()
	attempt := 0
	for {
		if err := ctx.Err(); err != nil {
			return sdr.Stream{}, false
		}

		attempt++
		device, stream, identity, err := d.openDevice()
		if err == nil {
			// A shutdown may have landed between acquisition and the swap;
			// never leave a freshly opened device behind.
			if ctxErr := ctx.Err(); ctxErr != nil {
				_ = device.Close()
				return sdr.Stream{}, false
			}
			d.sdr = device
			d.deviceIdentity = identity
			d.notifyAcquired(false, attempt, prevProgress)

			elapsed := d.now().Sub(startedAt).Round(time.Millisecond)
			d.logger.Info("RTL-SDR re-acquired",
				"elapsed", elapsed, "device", identity, "attempt", attempt)
			return stream, true
		}

		wait := recoveryBackoffDelay(attempt, capDelay)
		d.logger.Info("RTL-SDR recovery attempt failed; retrying",
			"attempt", attempt, "retry_in", wait, "error", err)
		if sleepErr := d.sleeper(ctx, wait); sleepErr != nil {
			return sdr.Stream{}, false
		}
	}
}

// startStream begins async sample delivery sized for one Decode call per
// block.
func (d *Decoder) startStream(device sdr.SDR) (sdr.Stream, error) {
	if d.decoder.Cfg.BlockSize2 <= 0 {
		return sdr.Stream{}, fmt.Errorf("%w: %d", ErrInvalidBlockSize, d.decoder.Cfg.BlockSize2)
	}
	//nolint:gosec // BlockSize2 is a positive power-of-two byte count
	bufLen := uint32(d.decoder.Cfg.BlockSize2)

	// Begin async sample delivery; blocks are dropped if the decode loop stalls.
	stream, err := device.StartStreaming(bufLen, 0)
	if err != nil {
		return sdr.Stream{}, fmt.Errorf("failed to start sample streaming: %w", err)
	}

	return stream, nil
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
