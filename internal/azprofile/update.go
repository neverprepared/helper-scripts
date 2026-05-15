package azprofile

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/neverprepared/azprofile/internal/ui"
)

const (
	githubReleasesURL = "https://api.github.com/repos/neverprepared/azprofile/releases/latest"
	httpUserAgent     = "azprofile-self-updater"
)

// UpdateOptions controls the update flow.
type UpdateOptions struct {
	Check bool // print status and exit; never download or replace
	Yes   bool // skip the interactive confirmation prompt
	Force bool // override dev-build refusal and same-version short-circuit
}

type ghAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

type ghRelease struct {
	TagName string    `json:"tag_name"`
	Body    string    `json:"body"`
	Assets  []ghAsset `json:"assets"`
}

// Update fetches the latest release from GitHub and (with confirmation) replaces
// the running binary in place. Returns nil on success or when --check is set.
func Update(opts UpdateOptions) error {
	rel, err := fetchLatestRelease()
	if err != nil {
		return fmt.Errorf("fetch latest release: %w", err)
	}

	cmp, cmpErr := compareSemver(Version, rel.TagName)

	if opts.Check {
		switch {
		case cmpErr != nil:
			fmt.Printf("%s-%s current=%s latest=%s (cannot compare)%s\n",
				ui.Dim, ui.NC, Version, rel.TagName, ui.NC)
		case cmp == 0:
			fmt.Printf("%s%s%s Up to date (%s)\n", ui.Green, ui.Check, ui.NC, Version)
		case cmp < 0:
			fmt.Printf("%s%s%s Update available: %s %s %s\n",
				ui.Yellow, ui.Arrow, ui.NC, Version, ui.Arrow, rel.TagName)
		case cmp > 0:
			fmt.Printf("%s-%s current=%s is newer than latest release %s%s\n",
				ui.Dim, ui.NC, Version, rel.TagName, ui.NC)
		}
		return nil
	}

	if cmpErr != nil && !opts.Force {
		return fmt.Errorf("current version %q is not a release version (dev build?); use `make install` to update locally, or pass --force", Version)
	}
	if cmpErr == nil {
		if cmp == 0 && !opts.Force {
			fmt.Printf("%s%s%s Already on %s (pass --force to reinstall)\n",
				ui.Green, ui.Check, ui.NC, Version)
			return nil
		}
		if cmp > 0 && !opts.Force {
			return fmt.Errorf("current version %s is newer than latest release %s; pass --force to downgrade", Version, rel.TagName)
		}
	}

	if isDevInstall() && !opts.Force {
		return fmt.Errorf("this binary appears to be a dev build (sibling .git/Makefile/go.mod found); use `make install` to update, or pass --force")
	}

	if !opts.Yes {
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			return fmt.Errorf("non-interactive run requires --yes")
		}
		fmt.Printf("\n%s%s%s Update %s %s %s\n",
			ui.Cyan, ui.Arrow, ui.NC, Version, ui.Arrow, rel.TagName)
		printReleaseNotesExcerpt(rel.Body)
		fmt.Print("\nProceed? [y/N] ")
		var resp string
		_, _ = fmt.Scanln(&resp)
		if strings.TrimSpace(strings.ToLower(resp)) != "y" {
			return fmt.Errorf("cancelled")
		}
	}

	return downloadAndReplace(rel)
}

func fetchLatestRelease() (*ghRelease, error) {
	req, err := http.NewRequest("GET", githubReleasesURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", httpUserAgent)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("GitHub API HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	if rel.TagName == "" {
		return nil, fmt.Errorf("GitHub API returned no tag name")
	}
	return &rel, nil
}

var semverRE = regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)$`)

// compareSemver returns -1, 0, +1 comparing a vs b. Errors if either isn't vX.Y.Z.
func compareSemver(a, b string) (int, error) {
	am := semverRE.FindStringSubmatch(a)
	bm := semverRE.FindStringSubmatch(b)
	if am == nil {
		return 0, fmt.Errorf("not semver: %q", a)
	}
	if bm == nil {
		return 0, fmt.Errorf("not semver: %q", b)
	}
	for i := 1; i <= 3; i++ {
		ai, _ := strconv.Atoi(am[i])
		bi, _ := strconv.Atoi(bm[i])
		if ai < bi {
			return -1, nil
		}
		if ai > bi {
			return 1, nil
		}
	}
	return 0, nil
}

// isDevInstall returns true when the binary's directory (or any ancestor up to
// 5 levels) contains a .git/go.mod/Makefile marker, indicating a checked-out
// workspace rather than a release install.
func isDevInstall() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	dir := filepath.Dir(exe)
	for i := 0; i < 5; i++ {
		for _, marker := range []string{".git", "go.mod", "Makefile"} {
			if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
				return true
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return false
		}
		dir = parent
	}
	return false
}

func printReleaseNotesExcerpt(body string) {
	body = strings.TrimSpace(body)
	if body == "" {
		return
	}
	lines := strings.Split(body, "\n")
	const maxLines = 10
	truncated := false
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		truncated = true
	}
	fmt.Printf("\n%s──── Release notes ────%s\n", ui.Dim, ui.NC)
	for _, l := range lines {
		fmt.Printf("%s  %s%s\n", ui.Dim, strings.TrimRight(l, "\r"), ui.NC)
	}
	if truncated {
		fmt.Printf("%s  ...%s\n", ui.Dim, ui.NC)
	}
}

func downloadAndReplace(rel *ghRelease) error {
	tag := rel.TagName
	tarballName := fmt.Sprintf("azprofile-%s-%s-%s.tar.gz", tag, runtime.GOOS, runtime.GOARCH)
	checksumName := fmt.Sprintf("azprofile-%s-checksums.txt", tag)
	tarballURL := assetURL(rel.Assets, tarballName)
	checksumURL := assetURL(rel.Assets, checksumName)
	if tarballURL == "" {
		return fmt.Errorf("no asset matching %s in release %s (unsupported platform?)", tarballName, tag)
	}
	if checksumURL == "" {
		return fmt.Errorf("no checksum file %s in release %s", checksumName, tag)
	}

	tmpDir, err := os.MkdirTemp("", "azprofile-update-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	tarballPath := filepath.Join(tmpDir, tarballName)
	fmt.Printf("%s%s%s Downloading %s\n", ui.Cyan, ui.Arrow, ui.NC, tarballName)
	if err := downloadFile(tarballURL, tarballPath); err != nil {
		return fmt.Errorf("download tarball: %w", err)
	}
	checksumPath := filepath.Join(tmpDir, checksumName)
	if err := downloadFile(checksumURL, checksumPath); err != nil {
		return fmt.Errorf("download checksums: %w", err)
	}

	fmt.Printf("%s%s%s Verifying SHA256\n", ui.Cyan, ui.Arrow, ui.NC)
	if err := verifySHA256(tarballPath, tarballName, checksumPath); err != nil {
		return err
	}

	fmt.Printf("%s%s%s Extracting\n", ui.Cyan, ui.Arrow, ui.NC)
	binPath, err := extractBinary(tarballPath, tmpDir)
	if err != nil {
		return err
	}

	exe, err := os.Executable()
	if err != nil {
		return err
	}
	target, err := filepath.EvalSymlinks(exe)
	if err != nil {
		target = exe
	}

	// Stage the replacement as a sibling of the target so the os.Rename below
	// is a same-filesystem (atomic) rename — even when tmpDir is on tmpfs.
	newPath := target + ".new"
	if err := copyFile(binPath, newPath, 0o755); err != nil {
		return fmt.Errorf("write %s: %w (do you have write permission?)", newPath, err)
	}
	if err := os.Rename(newPath, target); err != nil {
		_ = os.Remove(newPath)
		return fmt.Errorf("replace %s: %w", target, err)
	}

	fmt.Printf("\n%s%s%s Updated to %s%s%s\n", ui.Green, ui.Check, ui.NC, ui.Bold, tag, ui.NC)
	fmt.Printf("%s  Replaced: %s%s\n", ui.Dim, target, ui.NC)
	fmt.Printf("%s  Run `azprofile --version` to verify.%s\n", ui.Dim, ui.NC)
	return nil
}

func assetURL(assets []ghAsset, name string) string {
	for _, a := range assets {
		if a.Name == name {
			return a.BrowserDownloadURL
		}
	}
	return ""
}

func downloadFile(url, dst string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", httpUserAgent)
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d downloading %s", resp.StatusCode, url)
	}
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		return err
	}
	return nil
}

func verifySHA256(filePath, expectedName, checksumFile string) error {
	cb, err := os.ReadFile(checksumFile)
	if err != nil {
		return err
	}
	var want string
	for _, line := range strings.Split(string(cb), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// `shasum -a 256` format: "<hex>  <name>" (filename may be prefixed `*`).
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		n := strings.TrimPrefix(fields[len(fields)-1], "*")
		if n == expectedName {
			want = fields[0]
			break
		}
	}
	if want == "" {
		return fmt.Errorf("checksum for %s not found in checksum file", expectedName)
	}

	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	got := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(got, want) {
		return fmt.Errorf("SHA256 mismatch for %s: got %s want %s", expectedName, got, want)
	}
	return nil
}

func extractBinary(tarballPath, destDir string) (string, error) {
	f, err := os.Open(tarballPath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		// Basic tarbomb defense — refuse entries that try to escape with `..`
		// or absolute paths even though we only honor matches by basename.
		clean := filepath.Clean(hdr.Name)
		if strings.Contains(clean, "..") || filepath.IsAbs(clean) {
			return "", fmt.Errorf("tar entry escapes archive: %s", hdr.Name)
		}
		if filepath.Base(hdr.Name) != "azprofile" {
			continue
		}
		out := filepath.Join(destDir, "azprofile")
		of, err := os.OpenFile(out, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(of, tr); err != nil {
			of.Close()
			return "", err
		}
		if err := of.Close(); err != nil {
			return "", err
		}
		return out, nil
	}
	return "", fmt.Errorf("no `azprofile` binary found in tarball")
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
