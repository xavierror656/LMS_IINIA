package runner

import (
	"context"
	"testing"
	"time"
)

func TestMockNeverEvaluatesAndFinishes(t *testing.T) {
	var outputs []Output
	MockRunner{Delay: time.Millisecond}.Run(context.Background(), Request{Language: "javascript", Code: "throw new Error('EXECUTED')"}, func(o Output) bool { outputs = append(outputs, o); return true })
	if len(outputs) != 3 || outputs[2].Status != "completed" {
		t.Fatalf("unexpected outputs: %+v", outputs)
	}
	if outputs[0].Text != "Ejecución simulada: tu código no se ejecuta." {
		t.Fatal("simulation label missing")
	}
}
func TestMockCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	done := make(chan Output, 1)
	go MockRunner{Delay: time.Hour}.Run(ctx, Request{}, func(o Output) bool { done <- o; return true })
	select {
	case o := <-done:
		if o.Status != "cancelled" {
			t.Fatal(o)
		}
	case <-time.After(time.Second):
		t.Fatal("cancellation leaked runner")
	}
}
