package scrape

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"geo_client2/backend/config"
)

type BundleInfo struct {
	Version    string `json:"version"`
	Active     bool   `json:"active"`
	Size       int64  `json:"size"`
	UploadedAt string `json:"uploaded_at"`
}

type Manifest struct {
	BundleID      string         `json:"bundle_id"`
	BundleVersion string         `json:"bundle_version"`
	SchemaVersion int            `json:"schema_version"`
	Files         []ManifestFile `json:"files"`
}

type ManifestFile struct {
	Platform string `json:"platform"`
	Path     string `json:"path"`
	Sha256   string `json:"sha256"`
}

type ScrapeFlow struct {
	SchemaVersion int                      `json:"schemaVersion"`
	Platform      string                   `json:"platform"`
	Pipeline      []map[string]interface{} `json:"pipeline"`
}

type FlowResult struct {
	Queries   []string       `json:"queries"`
	Citations []FlowCitation `json:"citations"`
	FullText  string         `json:"full_text"`
}

type FlowCitation struct {
	URL          string `json:"url"`
	Domain       string `json:"domain"`
	Title        string `json:"title"`
	Snippet      string `json:"snippet"`
	QueryIndexes []int  `json:"query_indexes"`
	Query        string `json:"query"`
	CiteIndex    int    `json:"cite_index"`
}

type currentVersion struct {
	ActiveVersion string `json:"active_version"`
}

func bundleRoot() string {
	return filepath.Join(config.GetAppDir(), "flows", "scrape")
}

func bundlesDir() string {
	return filepath.Join(bundleRoot(), "bundles")
}

func currentPath() string {
	return filepath.Join(bundleRoot(), "current.json")
}

func ensureDir(path string) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", path, err)
	}
	return nil
}

func readActiveVersion() (string, error) {
	b, err := os.ReadFile(currentPath())
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("read current.json: %w", err)
	}
	var c currentVersion
	if err := json.Unmarshal(b, &c); err != nil {
		return "", fmt.Errorf("parse current.json: %w", err)
	}
	return strings.TrimSpace(c.ActiveVersion), nil
}

func writeActiveVersion(version string) error {
	if err := ensureDir(bundleRoot()); err != nil {
		return err
	}
	b, err := json.MarshalIndent(currentVersion{ActiveVersion: version}, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal current.json: %w", err)
	}
	tmp := currentPath() + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return fmt.Errorf("write current.json tmp: %w", err)
	}
	if err := os.Rename(tmp, currentPath()); err != nil {
		return fmt.Errorf("rename current.json: %w", err)
	}
	return nil
}

func ListBundles() ([]BundleInfo, string, error) {
	if err := ensureDir(bundlesDir()); err != nil {
		return nil, "", err
	}
	entries, err := os.ReadDir(bundlesDir())
	if err != nil {
		return nil, "", fmt.Errorf("read bundles dir: %w", err)
	}
	active, err := readActiveVersion()
	if err != nil {
		return nil, "", err
	}

	infos := make([]BundleInfo, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		version := e.Name()
		dir := filepath.Join(bundlesDir(), version)
		size, modTime, err := dirSizeAndTime(dir)
		if err != nil {
			return nil, "", err
		}
		infos = append(infos, BundleInfo{
			Version:    version,
			Active:     version == active,
			Size:       size,
			UploadedAt: modTime.Format(time.RFC3339),
		})
	}

	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Version > infos[j].Version
	})
	return infos, active, nil
}

func ImportBundle(base64Content string) (string, error) {
	if strings.TrimSpace(base64Content) == "" {
		return "", fmt.Errorf("empty bundle content")
	}
	data, err := base64.StdEncoding.DecodeString(base64Content)
	if err != nil {
		return "", fmt.Errorf("decode base64 bundle: %w", err)
	}

	tempDir, err := os.MkdirTemp("", "scrape_bundle_")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	if err := unzipToDir(data, tempDir); err != nil {
		return "", err
	}

	manifestPath := filepath.Join(tempDir, "manifest.json")
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return "", fmt.Errorf("read manifest.json: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return "", fmt.Errorf("parse manifest.json: %w", err)
	}
	if err := validateManifest(&manifest); err != nil {
		return "", err
	}

	for _, file := range manifest.Files {
		flowPath := filepath.Join(tempDir, filepath.FromSlash(file.Path))
		if err := validateFlowFile(flowPath, file); err != nil {
			return "", err
		}
	}

	version := time.Now().Format("20060102150405")
	bundleDir := filepath.Join(bundlesDir(), version)
	tmpBundle := filepath.Join(bundlesDir(), "._tmp_"+version)

	if err := ensureDir(bundlesDir()); err != nil {
		return "", err
	}
	if _, err := os.Stat(bundleDir); err == nil {
		return "", fmt.Errorf("bundle version already exists: %s", version)
	}
	if err := os.RemoveAll(tmpBundle); err != nil {
		return "", fmt.Errorf("cleanup temp bundle dir: %w", err)
	}
	if err := ensureDir(tmpBundle); err != nil {
		return "", err
	}

	if err := os.WriteFile(filepath.Join(tmpBundle, "manifest.json"), manifestBytes, 0o644); err != nil {
		return "", fmt.Errorf("write manifest.json: %w", err)
	}

	for _, file := range manifest.Files {
		src := filepath.Join(tempDir, filepath.FromSlash(file.Path))
		dst := filepath.Join(tmpBundle, filepath.FromSlash(file.Path))
		if err := ensureDir(filepath.Dir(dst)); err != nil {
			return "", err
		}
		if err := copyFile(src, dst); err != nil {
			return "", err
		}
	}

	if err := os.Rename(tmpBundle, bundleDir); err != nil {
		return "", fmt.Errorf("commit bundle dir: %w", err)
	}

	if err := pruneBundles(10); err != nil {
		return "", err
	}

	if err := writeActiveVersion(version); err != nil {
		return "", err
	}
	return version, nil
}

func pruneBundles(keep int) error {
	if keep <= 0 {
		return nil
	}
	entries, err := os.ReadDir(bundlesDir())
	if err != nil {
		return fmt.Errorf("read bundles dir: %w", err)
	}
	versions := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			versions = append(versions, e.Name())
		}
	}
	sort.Slice(versions, func(i, j int) bool { return versions[i] > versions[j] })
	if len(versions) <= keep {
		return nil
	}
	active, err := readActiveVersion()
	if err != nil {
		return err
	}
	for _, v := range versions[keep:] {
		if v == active {
			continue
		}
		_ = os.RemoveAll(filepath.Join(bundlesDir(), v))
	}
	return nil
}

func ExportBundle(version string) (string, error) {
	version = strings.TrimSpace(version)
	if version == "" {
		return "", fmt.Errorf("empty version")
	}
	dir := filepath.Join(bundlesDir(), version)
	if _, err := os.Stat(dir); err != nil {
		return "", fmt.Errorf("bundle not found: %s", version)
	}

	zipData, err := zipDir(dir)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(zipData), nil
}

func ExportActiveBundle() (string, string, error) {
	active, err := readActiveVersion()
	if err != nil {
		return "", "", err
	}
	if strings.TrimSpace(active) == "" {
		return "", "", fmt.Errorf("no active bundle")
	}
	content, err := ExportBundle(active)
	if err != nil {
		return "", "", err
	}
	return active, content, nil
}

func SwitchBundle(version string) error {
	version = strings.TrimSpace(version)
	if version == "" {
		return fmt.Errorf("empty version")
	}
	dir := filepath.Join(bundlesDir(), version)
	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("bundle not found: %s", version)
	}
	return writeActiveVersion(version)
}

func DeleteBundle(version string) error {
	version = strings.TrimSpace(version)
	if version == "" {
		return fmt.Errorf("empty version")
	}
	active, err := readActiveVersion()
	if err != nil {
		return err
	}
	if version == active {
		return fmt.Errorf("cannot delete active bundle")
	}
	dir := filepath.Join(bundlesDir(), version)
	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("bundle not found: %s", version)
	}
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("delete bundle dir: %w", err)
	}
	return nil
}

func validateManifest(m *Manifest) error {
	if m == nil {
		return fmt.Errorf("nil manifest")
	}
	if m.SchemaVersion != 1 {
		return fmt.Errorf("unsupported manifest schema_version=%d", m.SchemaVersion)
	}
	if len(m.Files) == 0 {
		return fmt.Errorf("manifest files empty")
	}
	for i, f := range m.Files {
		if strings.TrimSpace(f.Platform) == "" {
			return fmt.Errorf("manifest files[%d] missing platform", i)
		}
		if strings.TrimSpace(f.Path) == "" {
			return fmt.Errorf("manifest files[%d] missing path", i)
		}
		if !strings.HasPrefix(filepath.ToSlash(f.Path), "flows/") {
			return fmt.Errorf("manifest files[%d] path must start with flows/", i)
		}
	}
	return nil
}

func validateFlowFile(path string, meta ManifestFile) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read flow %s: %w", path, err)
	}
	if meta.Sha256 != "" {
		sum := sha256.Sum256(b)
		if !strings.EqualFold(meta.Sha256, hex.EncodeToString(sum[:])) {
			return fmt.Errorf("flow sha256 mismatch for %s", meta.Path)
		}
	}

	var flow ScrapeFlow
	if err := json.Unmarshal(b, &flow); err != nil {
		return fmt.Errorf("parse flow %s: %w", meta.Path, err)
	}
	if flow.SchemaVersion != 1 {
		return fmt.Errorf("flow schemaVersion unsupported for %s", meta.Path)
	}
	if strings.TrimSpace(flow.Platform) == "" {
		flow.Platform = meta.Platform
	}
	if flow.Platform != meta.Platform {
		return fmt.Errorf("flow platform mismatch for %s: want=%s got=%s", meta.Path, meta.Platform, flow.Platform)
	}
	if len(flow.Pipeline) == 0 {
		return fmt.Errorf("flow pipeline empty for %s", meta.Path)
	}
	if err := validateFlowActions(flow.Pipeline); err != nil {
		return fmt.Errorf("flow action validation failed for %s: %w", meta.Path, err)
	}
	return nil
}

func validateFlowActions(pipeline []map[string]interface{}) error {
	allowed := map[string]bool{
		"navigate":         true,
		"wait_load":        true,
		"wait_idle":        true,
		"wait":             true,
		"wait_ms":          true,
		"wait_selector":    true,
		"click":            true,
		"click_r":          true,
		"fill":             true,
		"frame_fill":       true,
		"eval":             true,
		"set_files":        true,
		"wait_url_not":     true,
		"wait_url_change":  true,
		"download_to_temp": true,
		"dom_extract":      true,
		"normalize":        true,
		"parse_json":       true,
		"set_result":       true,
		"needs_manual":     true,
	}
	for i, step := range pipeline {
		actionRaw, ok := step["action"]
		if !ok {
			return fmt.Errorf("step[%d] missing action", i)
		}
		action := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", actionRaw)))
		if action == "" {
			return fmt.Errorf("step[%d] empty action", i)
		}
		if !allowed[action] {
			return fmt.Errorf("step[%d] unsupported action=%s", i, action)
		}
	}
	return nil
}

func unzipToDir(data []byte, dest string) error {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	for _, f := range reader.File {
		cleaned, err := safeJoin(dest, f.Name)
		if err != nil {
			return err
		}
		if f.FileInfo().IsDir() {
			if err := ensureDir(cleaned); err != nil {
				return err
			}
			continue
		}
		if err := ensureDir(filepath.Dir(cleaned)); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("open zip file %s: %w", f.Name, err)
		}
		out, err := os.Create(cleaned)
		if err != nil {
			rc.Close()
			return fmt.Errorf("create file %s: %w", cleaned, err)
		}
		if _, err := io.Copy(out, rc); err != nil {
			out.Close()
			rc.Close()
			return fmt.Errorf("write file %s: %w", cleaned, err)
		}
		out.Close()
		rc.Close()
	}
	return nil
}

func safeJoin(base, name string) (string, error) {
	cleaned := filepath.Clean(name)
	if strings.Contains(cleaned, "..") || filepath.IsAbs(cleaned) {
		return "", fmt.Errorf("invalid zip path: %s", name)
	}
	return filepath.Join(base, cleaned), nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open src %s: %w", src, err)
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create dst %s: %w", dst, err)
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy %s -> %s: %w", src, dst, err)
	}
	return nil
}

func zipDir(dir string) ([]byte, error) {
	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		info, err := d.Info()
		if err != nil {
			return err
		}
		fh, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		fh.Name = rel
		fh.Method = zip.Deflate
		w, err := zw.CreateHeader(fh)
		if err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		if _, err := io.Copy(w, f); err != nil {
			f.Close()
			return err
		}
		f.Close()
		return nil
	})
	if err != nil {
		_ = zw.Close()
		return nil, fmt.Errorf("zip dir: %w", err)
	}
	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("close zip: %w", err)
	}
	return buf.Bytes(), nil
}

func dirSizeAndTime(dir string) (int64, time.Time, error) {
	var total int64
	mod := time.Time{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.ModTime().After(mod) {
			mod = info.ModTime()
		}
		if info.Mode().IsRegular() {
			total += info.Size()
		}
		return nil
	})
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("walk bundle dir: %w", err)
	}
	if mod.IsZero() {
		mod = time.Now()
	}
	return total, mod, nil
}
