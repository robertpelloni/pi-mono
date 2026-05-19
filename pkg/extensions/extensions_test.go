package extensions

import (
	"testing"
)

func TestRegistry_Exists(t *testing.T) {
	if len(Registry) == 0 {
		t.Error("Expected non-empty registry")
	}
}

func TestRegistry_AllExtensionsDisabled(t *testing.T) {
	for name, ext := range Registry {
		if ext.Enabled {
			t.Errorf("Extension %q should be disabled by default", name)
		}
	}
}

func TestRegistry_AllExtensionsHaveNames(t *testing.T) {
	for name, ext := range Registry {
		if ext.Name != name {
			t.Errorf("Extension key %q doesn't match Name %q", name, ext.Name)
		}
	}
}

func TestRegistry_AllExtensionsHaveDescriptions(t *testing.T) {
	for name, ext := range Registry {
		if ext.Description == "" {
			t.Errorf("Extension %q should have a description", name)
		}
	}
}

func TestRegistry_AllExtensionsHaveEnableFunc(t *testing.T) {
	for name, ext := range Registry {
		if ext.EnableFunc == nil {
			t.Errorf("Extension %q should have an EnableFunc", name)
		}
	}
}

func TestRegistry_EnableFuncsDontPanic(t *testing.T) {
	for name, ext := range Registry {
		if ext.EnableFunc != nil {
			err := ext.EnableFunc()
			if err != nil {
				t.Errorf("Extension %q EnableFunc returned error: %v", name, err)
			}
		}
	}
}

func TestRegistry_SpecificExtensions(t *testing.T) {
	expected := []string{
		"pi-better-ctx",
		"pi-rewind-hook",
		"pi-worktrees",
		"babysitter-pi",
		"pi-acp",
	}
	for _, name := range expected {
		if _, ok := Registry[name]; !ok {
			t.Errorf("Expected extension %q in registry", name)
		}
	}
}

func TestExtension_Fields(t *testing.T) {
	ext := Extension{
		Name:        "test",
		Enabled:     true,
		Description: "test extension",
		EnableFunc:  func() error { return nil },
	}
	if ext.Name != "test" || !ext.Enabled || ext.Description != "test extension" {
		t.Error("Field mismatch")
	}
}
