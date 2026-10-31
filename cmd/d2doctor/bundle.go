package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"d2lod-bot/internal/buildinfo"
	"d2lod-bot/internal/paths"
	"d2lod-bot/internal/setup"
)

const bundleLogLines = 500

// writeBundle writes a support bundle (tar.gz) to out: doctor.json, version.txt, the redacted config and the
// tails of the d2bot (instances/*/logs/d2bot.log) and d2fleet (logs/*fleet*.log, U: name not fixed by the fleet
// code) logs. Token files are never read.
func writeBundle(out string, rep setup.Report, l paths.Layout, now time.Time) error {
	var files []bundleFile
	doc, _ := json.MarshalIndent(rep, "", "  ")
	files = append(files,
		bundleFile{"doctor.json", doc},
		bundleFile{"version.txt", []byte(fmt.Sprintf("d2doctor %s\ncreated %s\n", buildinfo.Version, now.UTC().Format(time.RFC3339)))})
	if raw, err := os.ReadFile(l.Config()); err == nil {
		files = append(files, bundleFile{"config.redacted.json", redactConfig(raw)})
	}
	logs, _ := filepath.Glob(filepath.Join(l.InstancesDir(), "*", "logs", "d2bot.log"))
	fl, _ := filepath.Glob(filepath.Join(l.Logs(), "*fleet*.log"))
	for _, p := range append(logs, fl...) {
		rel, err := filepath.Rel(l.Root, p)
		if err != nil {
			continue
		}
		if t, err := tailLines(p, bundleLogLines); err == nil {
			files = append(files, bundleFile{"logs/" + filepath.ToSlash(rel), t})
		}
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, f := range files {
		if err := tw.WriteHeader(&tar.Header{Name: f.name, Mode: 0o644, Size: int64(len(f.data)), ModTime: now}); err != nil {
			return err
		}
		if _, err := tw.Write(f.data); err != nil {
			return err
		}
	}
	if err := tw.Close(); err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}
	return os.WriteFile(out, buf.Bytes(), 0o600)
}

type bundleFile struct {
	name string
	data []byte
}

// tailLines returns the last n lines of a file (it reads at most the last 4 MiB).
func tailLines(path string, n int) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	const max = 4 << 20
	off := int64(0)
	if st.Size() > max {
		off = st.Size() - max
	}
	b := make([]byte, st.Size()-off)
	if _, err := f.ReadAt(b, off); err != nil && len(b) > 0 {
		return nil, err
	}
	b = bytes.TrimRight(b, "\n")
	lines := bytes.Split(b, []byte("\n"))
	if off > 0 {
		lines = lines[1:] // first line is cut
	}
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return append(bytes.Join(lines, []byte("\n")), '\n'), nil
}

// redactConfig drops every key whose name contains token, secret, password or key (any depth, case-insensitive)
// and returns indented JSON. Unparseable input yields a placeholder, never the raw text.
func redactConfig(raw []byte) []byte {
	var v any
	if json.Unmarshal(raw, &v) != nil {
		return []byte("{\"error\": \"config is not valid JSON, omitted\"}\n")
	}
	b, _ := json.MarshalIndent(redact(v), "", "  ")
	return append(b, '\n')
}

func redact(v any) any {
	switch x := v.(type) {
	case map[string]any:
		for k, e := range x {
			lk := strings.ToLower(k)
			if strings.Contains(lk, "token") || strings.Contains(lk, "secret") || strings.Contains(lk, "password") || strings.Contains(lk, "key") {
				delete(x, k)
				continue
			}
			x[k] = redact(e)
		}
	case []any:
		for i, e := range x {
			x[i] = redact(e)
		}
	}
	return v
}
