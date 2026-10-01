package honker

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

// TestStreamSubscriberPollsWhenTheWatcherIsGone: waitLoop returns for good on
// any unexpected code from honker_watcher_wait, and nothing wakes a stream
// subscriber after that. The poll fallback is what still delivers.
func TestStreamSubscriberPollsWhenTheWatcherIsGone(t *testing.T) {
	extPath := findExtension(t)
	prev := streamPollFallback
	streamPollFallback = 50 * time.Millisecond
	t.Cleanup(func() { streamPollFallback = prev })

	d, err := Open(filepath.Join(t.TempDir(), "poll.db"), extPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	s := d.Stream("messages")
	tail, err := s.Publish(map[string]any{"n": 0})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events := s.SubscribeFrom(ctx, tail)
	// Let the subscriber reach its idle wait, then kill the watcher under it.
	time.Sleep(100 * time.Millisecond)
	d.updates.stop()

	want, err := s.Publish(map[string]any{"n": 1})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case ev, ok := <-events:
		if !ok {
			t.Fatal("the subscription closed instead of delivering")
		}
		if ev.Offset != want {
			t.Fatalf("got offset %d, want %d", ev.Offset, want)
		}
	case <-time.After(time.Second):
		t.Fatal("nothing delivered within 1s of a publish with the commit watcher gone")
	}
	// The subscriber reads streamPollFallback; wait for it to exit before the
	// cleanup puts the old value back.
	cancel()
	for range events {
	}
}
