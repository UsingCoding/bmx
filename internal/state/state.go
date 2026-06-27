package state

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/BurntSushi/toml"

	"github.com/UsingCoding/bmx/internal/model"
)

const CurrentVersion = 1

type File struct {
	Version    int      `toml:"version"`
	ActiveList string   `toml:"active_list"`
	Apps       []Record `toml:"apps"`
}

type Record struct {
	Name      string `toml:"name"`
	Manager   string `toml:"manager"`
	Package   string `toml:"package"`
	Installed bool   `toml:"installed"`
}

func Load(path string) (File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return File{Version: CurrentVersion}, nil
		}
		return File{}, fmt.Errorf("read state %s: %w", path, err)
	}

	var st File
	if err := toml.Unmarshal(data, &st); err != nil {
		return File{}, fmt.Errorf("decode state %s: %w", path, err)
	}
	if st.Version == 0 {
		st.Version = CurrentVersion
	}

	return st, nil
}

func (f File) InstalledApps() []model.App {
	apps := make([]model.App, 0, len(f.Apps))
	for _, record := range f.Apps {
		if !record.Installed {
			continue
		}
		apps = append(apps, model.App{
			Name:    record.Name,
			Manager: record.Manager,
			Package: record.Package,
		})
	}

	return apps
}

func FromApps(activeList string, apps []model.App) File {
	sorted := append([]model.App(nil), apps...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})

	records := make([]Record, 0, len(sorted))
	for _, app := range sorted {
		records = append(records, Record{
			Name:      app.Name,
			Manager:   app.Manager,
			Package:   app.Package,
			Installed: true,
		})
	}

	return File{
		Version:    CurrentVersion,
		ActiveList: activeList,
		Apps:       records,
	}
}

func Write(path string, st File) error {
	if st.Version == 0 {
		st.Version = CurrentVersion
	}

	var buf bytes.Buffer
	buf.WriteString("version = ")
	buf.WriteString(strconv.Itoa(st.Version))
	buf.WriteByte('\n')
	if st.ActiveList != "" {
		buf.WriteString("active_list = ")
		buf.WriteString(strconv.Quote(st.ActiveList))
		buf.WriteString("\n")
	}
	if len(st.Apps) > 0 {
		buf.WriteByte('\n')
	}

	for idx, app := range st.Apps {
		if idx > 0 {
			buf.WriteByte('\n')
		}
		buf.WriteString("[[apps]]\n")
		buf.WriteString("name = ")
		buf.WriteString(strconv.Quote(app.Name))
		buf.WriteByte('\n')
		buf.WriteString("manager = ")
		buf.WriteString(strconv.Quote(app.Manager))
		buf.WriteByte('\n')
		buf.WriteString("package = ")
		buf.WriteString(strconv.Quote(app.Package))
		buf.WriteByte('\n')
		buf.WriteString("installed = true\n")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(path), ".bmx-state-*.toml")
	if err != nil {
		return fmt.Errorf("create state temp file: %w", err)
	}
	tmpName := tmpFile.Name()
	defer os.Remove(tmpName)

	if _, err := tmpFile.Write(buf.Bytes()); err != nil {
		tmpFile.Close()
		return fmt.Errorf("write state temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close state temp file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace state file: %w", err)
	}

	return nil
}
