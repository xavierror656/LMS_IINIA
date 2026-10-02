package runner

import (
	"context"
	"time"
)

type Request struct{ Language, Code string }
type Output struct{ Kind, Text, Status string }
type Runner interface {
	Run(context.Context, Request, func(Output) bool)
}
type MockRunner struct{ Delay time.Duration }

func (m MockRunner) Run(ctx context.Context, _ Request, emit func(Output) bool) {
	delay := m.Delay
	if delay <= 0 {
		delay = 600 * time.Millisecond
	}
	for _, line := range []string{"Ejecución simulada: tu código no se ejecuta.", "El laboratorio está conectado. Un runner aislado se incorporará más adelante."} {
		select {
		case <-ctx.Done():
			emit(Output{Kind: "finished", Status: "cancelled"})
			return
		case <-time.After(delay):
			if !emit(Output{Kind: "stdout", Text: line}) {
				return
			}
		}
	}
	emit(Output{Kind: "finished", Status: "completed"})
}
