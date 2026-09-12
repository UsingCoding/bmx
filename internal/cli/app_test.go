package cli

import (
	"io"
	"os"
	"testing"
)

func TestDefaultRegistrySupportsBrewAndBrewCask(t *testing.T) {
	t.Parallel()

	registry := defaultRegistry(io.Discard)
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

func TestUseStylesDisablesRegularFile(t *testing.T) {
	t.Parallel()

	out, err := os.CreateTemp(t.TempDir(), "output")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := out.Close(); err != nil {
			t.Error(err)
		}
	})

	if useStyles(out) {
		t.Fatal("useStyles() = true for regular file")
	}
}
