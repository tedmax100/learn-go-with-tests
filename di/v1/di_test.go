package main

import (
	"bytes"
	"fmt"
	"testing"
)

func TestGreet(t *testing.T) {
	buffer := bytes.Buffer{}
	Greet(&buffer, "Chris")

	got := buffer.String()
	want := "Hello, Chris"

	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func Greet1(name string) {
	fmt.Printf("Hello, %s", name)
}
