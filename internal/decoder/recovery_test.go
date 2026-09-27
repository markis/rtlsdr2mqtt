package decoder

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"rtlsdr2mqtt/internal/sdr"
)

const (
	// testUSBPathA models the dongle's original bus/device node.
	testUSBPathA = "usb-001:003"
	// testUSBPathB models the node a re-enumerated dongle appears at.
	testUSBPathB = "usb-001:004"
)

var (
	// errTestBusy simulates a busy-on-reopen acquisition failure.
	errTestBusy = errors.New("test device busy on reopen")
	// errTestHiccup is a synthetic non-loss stream error.
	errTestHiccup = errors.New("test transient stream hiccup")
)

// fakeAcquisition is one scripted Acquire outcome: either a device or an
// error. Once the script is exhausted the last step repeats, modeling a
// device that stays absent.
type fakeAcquisition struct {
	device   *mockSDR
	identity string
	err      error
}

// fakeAcquirer scripts USB (re-)enumeration for recovery tests.
type fakeAcquirer struct {
	mu         sync.Mutex
	script     []fakeAcquisition
	calls      int
	identities []string
}

func (f *fakeAcquirer) Acquire() (sdr.SDR, string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	step := f.script[len(f.script)-1]
	if f.calls <= len(f.script) {
		step = f.script[f.calls-1]
	}
	if step.err != nil {
		return nil, "", step.err
	}
	// Mirror the production source: the returned device is opened.
	if err := step.device.Open(); err != nil {
		return nil, "", err
	}
	identity := step.identity
	if identity == "" {
		identity = fmt.Sprintf("mock-%d", f.calls)
	}
	f.identities = append(f.identities, identity)
	return step.device, identity, nil
}

func (f *fakeAcquirer) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

// staticAcquirer always returns the same device, modeling a healthy dongle.
func staticAcquirer(device *mockSDR) *fakeAcquirer {
	return &fakeAcquirer{script: []fakeAcquisition{{device: device, identity: "mock"}}}
}

// fakeSleeper records backoff waits without sleeping; when block is set it
// waits there (or for ctx cancellation) instead of returning immediately.
type fakeSleeper struct {
	mu    sync.Mutex
	waits []time.Duration
	block chan struct{}
}

func (f *fakeSleeper) sleep(ctx context.Context, d time.Duration) error {
	f.mu.Lock()
	f.waits = append(f.waits, d)
	block := f.block
	f.mu.Unlock()
	if block == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-block:
		return nil
	}
}

func (f *fakeSleeper) recordedWaits() []time.Duration {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]time.Duration(nil), f.waits...)
}

// logRecord captures one slog record for assertions.
type logRecord struct {
	level slog.Level
	msg   string
	attrs map[string]any
}

// recordingHandler is an slog.Handler that stores records for assertions.
type recordingHandler struct {
	mu      sync.Mutex
	records []logRecord
}

func newRecordingHandler() *recordingHandler {
	return &recordingHandler{}
}

func (h *recordingHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *recordingHandler) Handle(_ context.Context, r slog.Record) error { //nolint:gocritic // slog.Handler requires Record by value
	return h.handleRecord(&r)
}

func (h *recordingHandler) handleRecord(r *slog.Record) error {
	attrs := make(map[string]any, r.NumAttrs())
	r.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.Any()
		return true
	})
	h.mu.Lock()
	h.records = append(h.records, logRecord{level: r.Level, msg: r.Message, attrs: attrs})
	h.mu.Unlock()
	return nil
}

func (h *recordingHandler) WithAttrs([]slog.Attr) slog.Handler { return h }

func (h *recordingHandler) WithGroup(string) slog.Handler { return h }

func (h *recordingHandler) withMessage(level slog.Level, msg string) []logRecord {
	h.mu.Lock()
	defer h.mu.Unlock()
	var out []logRecord
	for _, r := range h.records {
		if r.level == level && r.msg == msg {
			out = append(out, r)
		}
	}
	return out
}

// waitFor polls cond until true or the timeout elapses.
func waitFor(t *testing.T, what string, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

// attrValue renders a recorded log attribute for assertions (slog decodes
// integers as int64, so compare the rendered form).
func attrValue(rec logRecord, key string) string {
	return fmt.Sprint(rec.attrs[key])
}

// pushProgress delivers one sample block so the session counts as having
// streamed (any block arrival marks decode progress).
func pushProgress(t *testing.T, d *Decoder, mock *mockSDR) {
	t.Helper()
	waitFor(t, "initial streaming", 2*time.Second, mock.isStreaming)
	mock.push(make([]byte, d.decoder.Cfg.BlockSize2))
	time.Sleep(50 * time.Millisecond)
}

func TestRecoveryBackoffDelay(t *testing.T) {
	tests := []struct {
		name    string
		cap     time.Duration
		attempt int
		want    time.Duration
	}{
		{"first attempt waits 1s", 5 * time.Second, 1, time.Second},
		{"second waits 2s", 5 * time.Second, 2, 2 * time.Second},
		{"third waits 5s", 5 * time.Second, 3, 5 * time.Second},
		{"later attempts hold the cap", 5 * time.Second, 9, 5 * time.Second},
		{"cap 10 keeps full ladder", 10 * time.Second, 4, 10 * time.Second},
		{"cap 10 holds steady", 10 * time.Second, 20, 10 * time.Second},
		{"cap 3 truncates ladder", 3 * time.Second, 3, 3 * time.Second},
		{"cap 3 holds steady", 3 * time.Second, 7, 3 * time.Second},
		{"cap 1 polls every second", time.Second, 2, time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := recoveryBackoffDelay(tt.attempt, tt.cap); got != tt.want {
				t.Errorf("recoveryBackoffDelay(%d, %v) = %v, want %v", tt.attempt, tt.cap, got, tt.want)
			}
		})
	}
}

// lossErr builds a loss-class stream error like the one librtlsdr surfaces
// when the dongle drops off the bus mid-read.
func lossErr() error {
	return fmt.Errorf("async sample read failed: %w", sdr.ErrDeviceLost)
}

func TestRecoveryOnLossClosesOldDeviceAndReacquires(t *testing.T) {
	oldMock, newMock := &mockSDR{}, &mockSDR{}
	acquirer := &fakeAcquirer{script: []fakeAcquisition{
		{device: oldMock, identity: testUSBPathA},
		{device: newMock, identity: testUSBPathB},
	}}
	d, _, handler := newRecoveryDecoder(t, acquirer)

	var touches atomic.Int32
	d.SetOnDeviceAcquired(func() { touches.Add(1) })

	ensureDecoderStarted(t, d)
	defer func() { _ = d.Stop() }()
	pushProgress(t, d, oldMock)

	if touches.Load() != 1 {
		t.Fatalf("expected one health touch on initial acquisition, got %d", touches.Load())
	}

	// The dongle drops off the bus: the read loop dies with a loss-class
	// error and the sample channel closes.
	oldMock.failStream(lossErr())

	// Recovery must close the stale handle, re-enumerate, and resume on the
	// new device without a process restart.
	waitFor(t, "recovery on new device", 5*time.Second, newMock.isStreaming)

	if oldCloses := oldMock.closeCount(); oldCloses != 1 {
		t.Errorf("expected old device closed exactly once, got %d closes", oldCloses)
	}
	if oldMock.isOpen() {
		t.Error("expected old device left closed")
	}

	if newMock.freqCallCount() == 0 {
		t.Error("expected configuration reapplied on the re-acquired device")
	}

	if touches.Load() != 2 {
		t.Errorf("expected health touch on re-acquisition (total 2), got %d", touches.Load())
	}
	if !d.IsRunning() {
		t.Error("expected decoder running after recovery")
	}

	lost := handler.withMessage(slog.LevelWarn, "RTL-SDR lost; entering re-enumeration recovery")
	if len(lost) != 1 {
		t.Fatalf("expected one loss-transition WARN, got %d", len(lost))
	}
	reacquired := handler.withMessage(slog.LevelInfo, "RTL-SDR re-acquired")
	if len(reacquired) != 1 {
		t.Fatalf("expected one re-acquired INFO, got %d", len(reacquired))
	}
	if attrValue(reacquired[0], "device") != testUSBPathB {
		t.Errorf("re-acquired device = %v, want %s", reacquired[0].attrs["device"], testUSBPathB)
	}
	if attrValue(reacquired[0], "attempt") != "1" {
		t.Errorf("re-acquired attempt = %v, want 1", reacquired[0].attrs["attempt"])
	}
}

func TestRecoveryAbsentThenReappears(t *testing.T) {
	newMock := &mockSDR{}
	absent := fmt.Errorf("poll: %w", sdr.ErrDeviceAbsent)
	oldMock := &mockSDR{}
	acquirer := &fakeAcquirer{script: []fakeAcquisition{
		{device: oldMock, identity: testUSBPathA},
		{err: absent},
		{err: absent},
		{err: absent},
		{device: newMock, identity: "usb-001:005"},
	}}
	d, sleeper, handler := newRecoveryDecoder(t, acquirer)

	var touches atomic.Int32
	d.SetOnDeviceAcquired(func() { touches.Add(1) })

	ensureDecoderStarted(t, d)
	defer func() { _ = d.Stop() }()
	pushProgress(t, d, oldMock)

	oldMock.failStream(lossErr())
	waitFor(t, "recovery on re-enumerated device", 5*time.Second, newMock.isStreaming)

	// Three failed attempts with the default 5s cap: 1s, 2s, 5s waits.
	wantWaits := []time.Duration{time.Second, 2 * time.Second, 5 * time.Second}
	if got := sleeper.recordedWaits(); fmt.Sprint(got) != fmt.Sprint(wantWaits) {
		t.Errorf("recorded backoff waits = %v, want %v", got, wantWaits)
	}
	if got := acquirer.callCount(); got != 5 {
		t.Errorf("acquirer calls = %d, want 5 (initial + 4 recovery attempts)", got)
	}

	failed := handler.withMessage(slog.LevelInfo, "RTL-SDR recovery attempt failed; retrying")
	if len(failed) != 3 {
		t.Fatalf("expected 3 failed-attempt INFO lines, got %d", len(failed))
	}
	for i, rec := range failed {
		if attrValue(rec, "attempt") != strconv.Itoa(i+1) {
			t.Errorf("attempt %d logged as %v", i+1, rec.attrs["attempt"])
		}
	}

	reacquired := handler.withMessage(slog.LevelInfo, "RTL-SDR re-acquired")
	if len(reacquired) != 1 {
		t.Fatalf("expected 1 re-acquired INFO, got %d", len(reacquired))
	}
	if attrValue(reacquired[0], "attempt") != "4" {
		t.Errorf("re-acquired attempt = %v, want 4", reacquired[0].attrs["attempt"])
	}

	// Failed polls must not touch health: one touch at startup, one at success.
	if touches.Load() != 2 {
		t.Errorf("health touches = %d, want 2 (startup + re-acquisition)", touches.Load())
	}
}

func TestRecoveryBusyThenSuccess(t *testing.T) {
	newMock := &mockSDR{}
	oldMock := &mockSDR{}

	acquirer := &fakeAcquirer{script: []fakeAcquisition{
		{device: oldMock, identity: testUSBPathA},
		{err: errTestBusy},
		{device: newMock, identity: testUSBPathA},
	}}
	d, sleeper, _ := newRecoveryDecoder(t, acquirer)

	ensureDecoderStarted(t, d)
	defer func() { _ = d.Stop() }()
	pushProgress(t, d, oldMock)

	oldMock.failStream(lossErr())
	waitFor(t, "recovery after busy reopen", 5*time.Second, newMock.isStreaming)

	if got := acquirer.callCount(); got != 3 {
		t.Errorf("acquirer calls = %d, want 3 (initial + busy + success)", got)
	}
	if got := sleeper.recordedWaits(); len(got) != 1 || got[0] != time.Second {
		t.Errorf("recorded backoff waits = %v, want [1s]", got)
	}
	if oldCloses := oldMock.closeCount(); oldCloses != 1 {
		t.Errorf("expected stale handle closed once before reopen, got %d closes", oldCloses)
	}
}

func TestRecoveryConfigurationReapplied(t *testing.T) {
	newMock := &mockSDR{}
	oldMock := &mockSDR{}
	acquirer := &fakeAcquirer{script: []fakeAcquisition{
		{device: oldMock, identity: testUSBPathA},
		{device: newMock, identity: testUSBPathB},
	}}
	d, _, _ := newRecoveryDecoder(t, acquirer)

	ensureDecoderStarted(t, d)
	defer func() { _ = d.Stop() }()
	pushProgress(t, d, oldMock)

	oldMock.failStream(lossErr())
	waitFor(t, "recovery streaming", 5*time.Second, newMock.isStreaming)

	// The protocol decoder defaults (scm+, symbol length 72) drive the SDR
	// setup; recovery must reapply exactly the same startup configuration.
	wantFreq, wantRate := uint32(912600155), uint32(2359296)
	if newMock.centerFreq != wantFreq {
		t.Errorf("re-acquired center freq = %d, want %d", newMock.centerFreq, wantFreq)
	}
	if newMock.sampleRate != wantRate {
		t.Errorf("re-acquired sample rate = %d, want %d", newMock.sampleRate, wantRate)
	}
	if resets := newMock.resetCallCount(); resets != 1 {
		t.Errorf("re-acquired buffer resets = %d, want 1", resets)
	}
}

func TestRecoveryOldDeviceClosedBeforeReopen(t *testing.T) {
	var mu sync.Mutex
	var events []string
	record := func(format string, args ...any) {
		mu.Lock()
		events = append(events, fmt.Sprintf(format, args...))
		mu.Unlock()
	}

	oldMock, newMock := &mockSDR{}, &mockSDR{}
	call := 0
	d, _, _ := newRecoveryDecoder(t, &fakeAcquirer{})
	d.SetDeviceSource(&orderingAcquirer{
		record: record,
		next: func() (sdr.SDR, string, error) {
			call++
			if call == 1 {
				record("acquire:old")
				return oldMock, testUSBPathA, nil
			}
			record("acquire:new")
			return newMock, testUSBPathB, nil
		},
	})
	oldMock.onClose = func() { record("close:old") }
	newMock.onClose = func() { record("close:new") }

	ensureDecoderStarted(t, d)
	defer func() { _ = d.Stop() }()
	pushProgress(t, d, oldMock)

	oldMock.failStream(lossErr())
	waitFor(t, "recovery streaming", 5*time.Second, newMock.isStreaming)

	mu.Lock()
	got := append([]string(nil), events...)
	mu.Unlock()

	closeIdx, acquireIdx := -1, -1
	for i, e := range got {
		if e == "close:old" && closeIdx < 0 {
			closeIdx = i
		}
		if e == "acquire:new" && acquireIdx < 0 {
			acquireIdx = i
		}
	}
	if closeIdx < 0 {
		t.Fatalf("stale handle was never closed, events: %v", got)
	}
	if acquireIdx < 0 {
		t.Fatalf("replacement was never acquired, events: %v", got)
	}
	if closeIdx > acquireIdx {
		t.Fatalf("stale handle closed after reopen (close@%d, acquire@%d), events: %v",
			closeIdx, acquireIdx, got)
	}
}

// orderingAcquirer scripts Acquire outcomes through a closure for tests that
// need to interleave event recording with acquisition.
type orderingAcquirer struct {
	record func(string, ...any)
	next   func() (sdr.SDR, string, error)
}

func (o *orderingAcquirer) Acquire() (sdr.SDR, string, error) {
	return o.next()
}

func TestRecoveryCancellationTerminatesPromptly(t *testing.T) {
	oldMock := &mockSDR{}
	absent := fmt.Errorf("poll: %w", sdr.ErrDeviceAbsent)
	acquirer := &fakeAcquirer{script: []fakeAcquisition{
		{device: oldMock, identity: testUSBPathA},
		{err: absent}, // repeats forever: the device stays absent
	}}
	d, sleeper, _ := newRecoveryDecoder(t, acquirer)
	// Block the backoff wait until the decoder context is canceled, so the
	// test proves cancellation (not a timer) unblocks recovery.
	sleeper.block = make(chan struct{})

	ensureDecoderStarted(t, d)
	waitFor(t, "initial streaming", 2*time.Second, oldMock.isStreaming)

	oldMock.failStream(lossErr())

	// Recovery must reach the blocked backoff wait: one initial acquire plus
	// one failed recovery attempt.
	waitFor(t, "recovery blocked in backoff", 5*time.Second, func() bool {
		return acquirer.callCount() == 2
	})

	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		_ = d.Stop()
	}()

	select {
	case <-stopped:
	case <-time.After(10 * time.Second):
		t.Fatal("Stop did not return promptly while recovery was blocked")
	}

	// No further acquisition attempts may happen after cancellation.
	callsAfterStop := acquirer.callCount()
	time.Sleep(200 * time.Millisecond)
	if got := acquirer.callCount(); got != callsAfterStop {
		t.Errorf("acquirer called after cancellation: %d -> %d", callsAfterStop, got)
	}
	if d.IsRunning() {
		t.Error("expected decoder stopped after cancellation")
	}
}

func TestRecoveryNonLossErrorKeepsExistingSemantics(t *testing.T) {
	oldMock := &mockSDR{}
	acquirer := &fakeAcquirer{script: []fakeAcquisition{
		{device: oldMock, identity: testUSBPathA},
		{device: &mockSDR{}, identity: testUSBPathB},
	}}
	d, _, handler := newRecoveryDecoder(t, acquirer)
	d.watchdogTimeout = 10 * time.Second

	var touches atomic.Int32
	d.SetOnDeviceAcquired(func() { touches.Add(1) })

	ensureDecoderStarted(t, d)
	defer func() { _ = d.Stop() }()
	waitFor(t, "initial streaming", 2*time.Second, oldMock.isStreaming)

	// A non-loss stream error (synthetic here) must not be mistaken for USB
	// re-enumeration: it is forwarded on errChan and the decoder stops the
	// session, leaving recovery and the device untouched.
	oldMock.failStream(errTestHiccup)

	_, errChan, _ := d.Channels()
	select {
	case err := <-errChan:
		if sdr.IsDeviceLost(err) {
			t.Errorf("non-loss error classified as device loss: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("non-loss stream error was not forwarded to errChan")
	}

	// Give the loop a moment: no recovery attempt may follow.
	time.Sleep(200 * time.Millisecond)
	if got := acquirer.callCount(); got != 1 {
		t.Errorf("acquirer calls = %d, want 1 (no recovery for non-loss error)", got)
	}
	if len(handler.withMessage(slog.LevelWarn, "RTL-SDR lost; entering re-enumeration recovery")) != 0 {
		t.Error("non-loss error incorrectly entered re-enumeration recovery")
	}
	if oldCloses := oldMock.closeCount(); oldCloses != 0 {
		t.Errorf("decoder closed the device on a non-loss error (%d closes)", oldCloses)
	}
	if touches.Load() != 1 {
		t.Errorf("health touches = %d, want 1 (startup only)", touches.Load())
	}
}
