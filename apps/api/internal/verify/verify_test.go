package verify

import (
	"context"
	"testing"
	"time"

	"github.com/opspilot/opspilot/apps/api/internal/release"
	"github.com/opspilot/opspilot/apps/api/internal/telemetry"
)

type fakeWindows struct {
	calls int
	snap  telemetry.Snapshot
}

func (f *fakeWindows) ServiceWindow(context.Context, string, string, string) (telemetry.Snapshot, error) {
	f.calls++
	return f.snap, nil
}

type fakeStore struct {
	result  string
	resolve bool
	detail  string
}

func (f *fakeStore) CompletePaymentVerification(_ context.Context, _, result, detail string, resolve bool) error {
	f.result = result
	f.detail = detail
	f.resolve = resolve
	return nil
}

func (f *fakeStore) VerifyingPayments(context.Context) ([]string, error) { return nil, nil }

func TestWatcherResolvesAfterTwoHealthyWindowsOnTheGoodImage(t *testing.T) {
	store := &fakeStore{}
	metrics := &fakeWindows{snap: telemetry.Snapshot{Available: true, Requests: 40, Errors: 0, ErrorRate: 0, P95: 0.01}}
	watcher := &Watcher{
		Metrics:  metrics,
		Image:    func(context.Context) (string, error) { return release.GoodImage, nil },
		Store:    store,
		Interval: time.Millisecond,
		Deadline: time.Second,
		Need:     2,
	}
	watcher.run(context.Background(), "INC-REAL-test")
	if !store.resolve || store.result != "healthy" || metrics.calls < 2 {
		t.Fatalf("store=%#v calls=%d", store, metrics.calls)
	}
}

func TestWatcherDoesNotResolveWhileTheBadImageIsRunning(t *testing.T) {
	store := &fakeStore{}
	watcher := &Watcher{
		Metrics:  &fakeWindows{snap: telemetry.Snapshot{Available: true, Requests: 40, P95: 0.01}},
		Image:    func(context.Context) (string, error) { return release.BadImage, nil },
		Store:    store,
		Interval: time.Millisecond,
		Deadline: 5 * time.Millisecond,
		Need:     2,
	}
	watcher.run(context.Background(), "INC-REAL-test")
	if store.resolve || store.result != "failed" {
		t.Fatalf("%#v", store)
	}
}

func TestWatcherDoesNotResolveOnADirtyWindow(t *testing.T) {
	store := &fakeStore{}
	watcher := &Watcher{
		Metrics: &fakeWindows{snap: telemetry.Snapshot{Available: true, Requests: 40, Errors: 12, ErrorRate: 0.3, P95: 0.8}},
		Image:   func(context.Context) (string, error) { return release.GoodImage, nil },
		Store:   store, Interval: time.Millisecond, Deadline: 5 * time.Millisecond, Need: 2,
	}
	watcher.run(context.Background(), "INC-REAL-test")
	if store.resolve || store.result != "failed" {
		t.Fatalf("%#v", store)
	}
}
