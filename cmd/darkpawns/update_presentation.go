package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/zax0rz/darkpawns/internal/portable"
)

type releaseNotes struct {
	Tag    string `json:"tag_name"`
	Body   string `json:"body"`
	Assets []struct {
		Name string `json:"name"`
		Size int64  `json:"size"`
	} `json:"assets"`
}

func presentationRelease(ctx context.Context) (releaseNotes, error) {
	const endpoint = "https://api.github.com/repos/zax0rz/darkpawns/releases/latest"
	var notes releaseNotes
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return notes, err
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	client := &http.Client{Transport: transport, Timeout: 4 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return fmt.Errorf("release notes redirect refused") }}
	defer client.CloseIdleConnections()
	res, err := client.Do(req)
	if err != nil {
		return notes, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return notes, fmt.Errorf("release notes returned HTTP %d", res.StatusCode)
	}
	err = json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&notes)
	return notes, err
}

func showReleaseNotes(cmd *cobra.Command, notes releaseNotes, verifiedVersion string) error {
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), cliTitle.Render("Release notes")); err != nil {
		return err
	}
	if notes.Body == "" || strings.TrimPrefix(notes.Tag, "v") != strings.TrimPrefix(verifiedVersion, "v") {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "Release notes unavailable for this verified release.")
		return err
	}
	theme := styles.ASCIIStyleConfig
	accent := "#A8201A"
	if lipgloss.HasDarkBackground() {
		accent = "#CF4B42"
	}
	yes := true
	theme.Strong.BlockPrefix = ""
	theme.Strong.BlockSuffix = ""
	theme.Strong.Prefix = ""
	theme.Strong.Suffix = ""
	theme.Strong.Bold = &yes
	theme.Emph.BlockPrefix = ""
	theme.Emph.BlockSuffix = ""
	theme.Emph.Prefix = ""
	theme.Emph.Suffix = ""
	theme.Emph.Italic = &yes
	theme.Heading.Bold = &yes
	theme.Heading.Color = &accent
	theme.H1 = theme.Heading
	theme.H2 = theme.Heading
	theme.Link.Color = &accent
	theme.LinkText.Color = &accent
	renderer, err := glamour.NewTermRenderer(glamour.WithStyles(theme), glamour.WithWordWrap(max(20, cliWidth(cmd)-4)))
	if err != nil {
		return err
	}
	rendered, err := renderer.Render(terminalText(notes.Body))
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(cmd.OutOrStdout(), rendered)
	return err
}

type updateObserver struct {
	instance                                string
	before                                  map[string]bool
	size                                    int64
	prepared, applied, applying, wasRunning atomic.Bool
}

func newUpdateObserver(path string, size int64) *updateObserver {
	o := &updateObserver{instance: path, size: size, before: map[string]bool{}}
	entries, _ := os.ReadDir(filepath.Join(path, ".updates"))
	for _, e := range entries {
		o.before[e.Name()] = true
	}
	return o
}

func (o *updateObserver) phases() []phaseView {
	download := phaseView{name: "Download", detail: "Waiting for the verified archive; no guessed percentage"}
	verify := phaseView{name: "Verify", detail: "Publisher and bundle verification must return successfully"}
	stage := phaseView{name: "Stage", detail: "Waiting for the existing staging operation"}
	health := phaseView{name: "Health gate", detail: "Not started; the world keeps running during preparation", skip: !o.applying.Load()}
	entries, _ := os.ReadDir(filepath.Join(o.instance, ".updates"))
	for _, e := range entries {
		if !e.IsDir() || o.before[e.Name()] {
			continue
		}
		work := filepath.Join(o.instance, ".updates", e.Name())
		if info, err := os.Stat(filepath.Join(work, "download")); err == nil {
			download.detail = fmt.Sprintf("%s received", byteSize(info.Size()))
			if o.size > 0 {
				download.known = true
				download.fraction = min(1, float64(info.Size())/float64(o.size))
				download.detail += fmt.Sprintf(" / %s", byteSize(o.size))
			}
		}
		if _, err := os.Stat(filepath.Join(work, "release")); err == nil {
			download.complete = true
			download.detail = "Archive received; checking extracted content"
			verify.detail = "Extraction, signature checks and staging in progress"
		}
	}
	if o.prepared.Load() {
		download.complete = true
		verify.complete = true
		stage.complete = true
		download.detail = "Archive received"
		verify.detail = "Portable verification succeeded"
		stage.detail = "Candidate is ready; active world unchanged"
	}
	if o.applying.Load() {
		health.skip = false
		health.detail = "Saving, backing up and checking startup readiness"
		var journal struct {
			ID    string `json:"id"`
			Phase string `json:"phase"`
		}
		if readPresentationJSON(filepath.Join(o.instance, ".updates", "transaction.json"), &journal) == nil && journal.Phase != "" && journal.ID != "" && !o.before[journal.ID] {
			health.detail = "Transaction: " + journal.Phase + " · waiting for portable.Apply"
		}
	}
	if o.applied.Load() {
		if o.wasRunning.Load() {
			health.complete = true
			health.detail = "Portable startup health gate passed"
		} else {
			health.skip = true
			health.detail = "Not run: the instance was stopped before this update"
		}
	}
	return []phaseView{download, verify, stage, health}
}

func readPresentationJSON(path string, value any) error {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewDecoder(io.LimitReader(f, 128<<10)).Decode(value)
}

func byteSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	return fmt.Sprintf("%.1f MiB", float64(size)/(1<<20))
}

func runInteractiveUpdate(cmd *cobra.Command, path, base, archive, digest string, yes, check, allowDowngrade bool) (result error) {
	notes := releaseNotes{}
	size := int64(0)
	if archive != "" {
		if info, err := os.Stat(archive); err == nil {
			size = info.Size()
		}
	} else if os.Getenv("DP_RELEASE_ENDPOINT") == "" {
		// Private feeds remain entirely owned by portable; optional notes use only GitHub.
		// Notes and sizes are presentation hints, never inputs to trusted verification.
		var infoErr error
		notes, infoErr = runWait(cmd, "Reading release information", nil, func(ctx context.Context) (releaseNotes, error) {
			n, e := presentationRelease(ctx)
			if ctx.Err() != nil {
				return n, ctx.Err()
			}
			if e != nil {
				return releaseNotes{}, nil
			}
			return n, nil
		})
		if infoErr != nil {
			return infoErr
		}
	}
	observer := newUpdateObserver(path, size)
	plan, err := runWait(cmd, "Prepare update · your world keeps running", observer.phases, func(ctx context.Context) (*portable.UpdatePlan, error) {
		p, err := portable.PrepareUpdate(ctx, base, path, portable.UpdateOptions{Archive: archive, SHA256: digest, AllowDowngrade: allowDowngrade})
		if err == nil {
			observer.prepared.Store(true)
		}
		return p, err
	})
	if err != nil {
		if plan != nil {
			err = errors.Join(err, plan.Close())
		}
		return err
	}
	defer func() { result = errors.Join(result, plan.Close()) }()
	checks := []portable.Check{{Name: "Release " + plan.PreviousVersion + " → " + plan.Version, OK: true, Detail: "Verified candidate"}, {Name: "Archive SHA-256", OK: true, Detail: plan.SHA256}, {Name: "Active world", OK: true, Detail: "Not stopped during preparation"}}
	if err := checklist(cmd.OutOrStdout(), "Dark Pawns · release check", checks, cliWidth(cmd)); err != nil {
		return err
	}
	if err := showReleaseNotes(cmd, notes, plan.Version); err != nil {
		return err
	}
	if plan.UpToDate {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "Already up to date. Your world was not stopped.")
		return err
	}
	if check {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "Release verified. No update applied.")
		return err
	}
	status, err := portable.Status(cmd.Context(), path)
	if err != nil {
		return err
	}
	observer.wasRunning.Store(status.State == "running")
	if !yes {
		reader, err := signInAdmin(cmd, path, status)
		if err != nil {
			return err
		}
		if reader != nil {
			defer reader.client.CloseIdleConnections()
		}
		snapshot := readCLISnapshot(cmd.Context(), path, base, reader)
		approved, err := confirmAction(cmd, "Apply this update?", onlineSummary(snapshot)+"\nThis disconnects players. A private backup is retained. If startup fails, portable restores the previous snapshot and retains attempted state.")
		if err != nil {
			return err
		}
		if !approved {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), "Update cancelled. Your world was not stopped.")
			return err
		}
	}
	observer.applying.Store(true)
	_, err = runWait(cmd, "Apply update", observer.phases, func(ctx context.Context) (struct{}, error) {
		err := plan.Apply(ctx)
		if err == nil {
			observer.applied.Store(true)
		}
		return struct{}{}, err
	})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), "Update complete. Release and backup are retained in your instance's .updates folder.")
	return err
}
