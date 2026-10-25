// Command minimal is the smallest ori-agent application: an agent that
// reasons briefly, shows a tool call and echoes the prompt.
//
//	go run ./examples/minimal
package main

import (
	"context"
	"strings"
	"time"

	oriagent "github.com/sitle/ori-agent"
)

func main() {
	oriagent.Run(oriagent.Config{
		AppName: "Écho",
		Agent:   oriagent.AgentFunc(echo),
	})
}

func echo(ctx context.Context, t *oriagent.Turn) error {
	if t.Thinking {
		th := t.Think()
		th.Write("Je répète simplement le message reçu.")
		th.Done()
	}
	call := t.Tool("Count", "words")
	call.Done(strings.Repeat("·", len(strings.Fields(t.Prompt))))

	b := t.Text()
	defer b.Done()
	for _, w := range strings.SplitAfter("Vous avez écrit : "+t.Prompt, " ") {
		select {
		case <-ctx.Done(): // the user pressed Stop
			return ctx.Err()
		case <-time.After(40 * time.Millisecond):
		}
		b.Write(w)
	}
	return nil
}
