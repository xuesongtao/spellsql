package utils

import (
	"sync"
	"testing"
)

func TestInitCallOnce(t *testing.T) {
	called := 0
	fn := InitCallOnce(func() {
		called++
	})
	fn()
	fn()
	if called != 1 {
		t.Errorf("expected called to be 1, got %d", called)
	}
}

func TestInitCallOnce1(t *testing.T) {
	called := 0
	fn := InitCallOnce(func() {
		called++
	})
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		fn()
	}()
	go func() {
		defer wg.Done()
		fn()
	}()
	wg.Wait()
	if called != 1 {
		t.Errorf("expected called to be 1, got %d", called)
	}
}
