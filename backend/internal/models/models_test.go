package models

import (
	"encoding/json"
	"testing"
)

func TestLessonConfig(t *testing.T) {
	for _, l := range []Lesson{{Type: "reading", Config: json.RawMessage(`{}`)}, {Type: "code", Config: json.RawMessage(`{"language":"shell"}`)}, {Type: "h5p", Config: json.RawMessage(`{"activity":"https://evil.example"}`)}} {
		if l.Validate() == nil {
			t.Fatal("invalid config accepted")
		}
	}
}
