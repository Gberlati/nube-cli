package cmd

import "testing"

func TestConfirmDestructive_ForceSkips(t *testing.T) {
	flags := &RootFlags{Force: true}
	if err := confirmDestructive(flags, "delete all"); err != nil {
		t.Errorf("Force=true should skip, got error: %v", err)
	}
}

func TestConfirmDestructive_NoInputErrors(t *testing.T) {
	flags := &RootFlags{NoInput: true}
	err := confirmDestructive(flags, "delete all")
	if err == nil {
		t.Fatal("NoInput=true should error")
	}
}

func TestConfirmDestructive_NilFlags(t *testing.T) {
	if err := confirmDestructive(nil, "test"); err != nil {
		t.Errorf("nil flags should skip, got error: %v", err)
	}
}
