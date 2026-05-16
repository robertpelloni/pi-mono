package eventbus

import (
	"sync/atomic"
	"testing"
)

func TestEventBus_OnEmit(t *testing.T) {
	bus := New()
	var received atomic.Int32

	On(bus, "test", func(e string) {
		if e == "hello" {
			received.Add(1)
		}
	})

	Emit(bus, "test", "hello")
	if received.Load() != 1 {
		t.Errorf("Expected 1, got %d", received.Load())
	}
}

func TestEventBus_MultipleListeners(t *testing.T) {
	bus := New()
	var count atomic.Int32

	On(bus, "test", func(e int) {
		count.Add(1)
	})
	On(bus, "test", func(e int) {
		count.Add(1)
	})

	Emit(bus, "test", 42)
	if count.Load() != 2 {
		t.Errorf("Expected 2, got %d", count.Load())
	}
}

func TestEventBus_Unsubscribe(t *testing.T) {
	bus := New()
	var count atomic.Int32

	unsub := On(bus, "test", func(e string) {
		count.Add(1)
	})

	Emit(bus, "test", "a")
	if count.Load() != 1 {
		t.Errorf("Expected 1, got %d", count.Load())
	}

	unsub()
	Emit(bus, "test", "b")
	if count.Load() != 1 {
		t.Errorf("Expected 1 after unsubscribe, got %d", count.Load())
	}
}

func TestEventBus_Once(t *testing.T) {
	bus := New()
	var count atomic.Int32

	Once(bus, "test", func(e string) {
		count.Add(1)
	})

	Emit(bus, "test", "a")
	Emit(bus, "test", "b")
	if count.Load() != 1 {
		t.Errorf("Expected 1 from once listener, got %d", count.Load())
	}
}

func TestEventBus_DifferentTypes(t *testing.T) {
	bus := New()
	var intReceived atomic.Int32
	var strReceived atomic.Int32

	On(bus, "int-event", func(e int) {
		intReceived.Add(1)
	})
	On(bus, "str-event", func(e string) {
		strReceived.Add(1)
	})

	Emit(bus, "int-event", 42)
	Emit(bus, "str-event", "hello")

	if intReceived.Load() != 1 {
		t.Errorf("Expected 1 int event, got %d", intReceived.Load())
	}
	if strReceived.Load() != 1 {
		t.Errorf("Expected 1 str event, got %d", strReceived.Load())
	}
}

func TestEventBus_Clear(t *testing.T) {
	bus := New()
	On(bus, "test", func(e string) {})
	On(bus, "other", func(e int) {})

	if bus.ListenerCount("test") != 1 || bus.ListenerCount("other") != 1 {
		t.Error("Expected 1 listener each")
	}

	bus.Clear()
	if bus.ListenerCount("test") != 0 || bus.ListenerCount("other") != 0 {
		t.Error("Expected 0 listeners after clear")
	}
}
