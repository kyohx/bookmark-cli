package main

import "testing"

func TestBookmarkCommandsRegistered(t *testing.T) {
	for _, name := range []string{"list", "get", "add", "update", "delete"} {
		cmd, _, err := newRootCmd().Find([]string{name})
		if err != nil || cmd.Name() != name {
			t.Fatalf("command %s not registered: %v", name, err)
		}
	}
	cmd := newBookmarkCmd(&rootOptions{})
	for _, name := range []string{"list", "get", "add", "update", "delete"} {
		child, _, err := cmd.Find([]string{name})
		if err != nil || child.Name() != name {
			t.Fatalf("nested command %s not registered: %v", name, err)
		}
	}
}
