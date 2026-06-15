package rule

import (
	"context"
	"testing"
)

func TestExecutor_CSS(t *testing.T) {
	body := `<html><body><div class="item">Hello</div></body></html>`
	e := NewExecutor()
	out, err := e.Execute(context.Background(), ModeDefault, ".item", body)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0] != "Hello" {
		t.Fatalf("got %v", out)
	}
}
