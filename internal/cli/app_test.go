package cli

import "testing"

func TestDefaultRegistrySupportsBrewAndBrewCask(t *testing.T) {
	t.Parallel()

	registry := defaultRegistry()
	brewManager, ok := registry["brew"]
	if !ok {
		t.Fatal("brew manager missing")
	}

	brewCaskManager, ok := registry["brew-cask"]
	if !ok {
		t.Fatal("brew-cask manager missing")
	}

	if brewManager != brewCaskManager {
		t.Fatal("brew and brew-cask should share one backend manager")
	}
}
