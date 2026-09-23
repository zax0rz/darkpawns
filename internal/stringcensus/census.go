package stringcensus

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Defaults for the four things the census reads and the one it writes. They are
// paths relative to the repository root.
var (
	// DefaultGoDirs are the packages whose literals can reach a player.
	DefaultGoDirs = []string{"pkg/game", "pkg/session", "pkg/combat", "pkg/spells"}
	// DefaultCDir is the read-only C oracle (RULEBOOK R5: src/ is ground truth).
	DefaultCDir = "src"
	// DefaultDataDirs are the world data trees: data-driven text is
	// data-sourced, not invented (brief item 4).
	DefaultDataDirs = []string{"lib/world", "lib/misc", "lib/text"}
	// DefaultOutDir receives the generated reports.
	DefaultOutDir = "docs/fidelity/strings"
)

// Options configures one census run.
type Options struct {
	// Root is the repository root.
	Root string
	// GoDirs, CDir and DataDirs are relative to Root.
	GoDirs   []string
	CDir     string
	DataDirs []string
	// OutDir is where the reports are written, relative to Root.
	OutDir string
}

func (o Options) withDefaults() Options {
	if len(o.GoDirs) == 0 {
		o.GoDirs = DefaultGoDirs
	}
	if o.CDir == "" {
		o.CDir = DefaultCDir
	}
	if len(o.DataDirs) == 0 {
		o.DataDirs = DefaultDataDirs
	}
	if o.OutDir == "" {
		o.OutDir = DefaultOutDir
	}
	return o
}

// sourceFile is one file read for the census, with the path it is reported as.
type sourceFile struct {
	// rel is the path relative to the repository root, with forward slashes.
	rel string
	// data is the file contents.
	data string
}

// Segment is one normalized fixed-text segment with the source site that
// produced it.
type Segment struct {
	Text string
	File string
	Line int
	Sink string
	// Fn is the enclosing C function; empty on the Go side.
	Fn string
}

// candidate is one player-facing string found in a source tree, before
// normalization forks it into segments. It carries the raw (unescaped) literal
// so a report row can be traced back to the exact source text.
type candidate struct {
	file string
	line int
	sink string
	// fn is the enclosing function. The C side fills it; the Go side leaves it
	// empty, since a file:line finds the site.
	fn  string
	raw string
}

// UnresolvedSite is a Go sink argument the extractor could not resolve.
type UnresolvedSite struct {
	File string
	Line int
	Sink string
	Expr string
}

// Report is the result of one census run. The slices are sorted and
// deduplicated, so the same inputs always produce the same report.
type Report struct {
	GoFiles, CFiles, DataFiles int
	GoCandidates, CCandidates  int
	// GoSegmentCount and CSegmentCount are the normalized segment totals.
	GoSegmentCount, CSegmentCount int
	GoOnly, CMissing              int
	DataSourced                   int

	// GoOnlySegments are the segments with neither a C nor a world-data
	// source: the R4 review queue the ratchet watches.
	GoOnlySegments []Segment
	// CMissingSegments are C segments no Go segment contains: port gaps.
	CMissingSegments []Segment
	// DataSourcedSegments are Go-only-by-code segments found in the world data.
	DataSourcedSegments []Segment
	// UnresolvedSites are sink arguments the extractor refused to guess about.
	UnresolvedSites []UnresolvedSite

	// sinkCounts counts candidates per sink, per tree.
	goSinkCounts map[string]int
	cSinkCounts  map[string]int
	// goPkgCounts counts Go candidates per directory; cFileCounts counts C
	// candidates per C file.
	goPkgCounts map[string]int
	cFileCounts map[string]int
	// goOnlyByFile and cMissingByFile count reported segments per file.
	goOnlyByFile   map[string]int
	cMissingByFile map[string]int
}

func (r *Report) String() string {
	return fmt.Sprintf("go-only %d, c-missing %d, data-sourced %d, unresolved %d",
		r.GoOnly, r.CMissing, r.DataSourced, len(r.UnresolvedSites))
}

// Run performs the census described by opts.
func Run(opts Options) (*Report, error) {
	opts = opts.withDefaults()
	root, err := filepath.Abs(opts.Root)
	if err != nil {
		return nil, err
	}

	goSrc, err := readSourceFiles(root, opts.GoDirs, isGoSource)
	if err != nil {
		return nil, err
	}
	cSrc, err := readSourceFiles(root, []string{opts.CDir}, isCSource)
	if err != nil {
		return nil, err
	}
	dataSrc, err := readSourceFiles(root, opts.DataDirs, func(string, fs.DirEntry) bool { return true })
	if err != nil {
		return nil, err
	}

	goCands, unresolved, err := extractGoFiles(goSrc)
	if err != nil {
		return nil, err
	}
	cCands := extractCFiles(cSrc)
	return buildReport(goSrc, cSrc, dataSrc, goCands, cCands, unresolved), nil
}

// readSourceFiles walks each directory under root and reads the files keep
// accepts, reporting paths relative to root.
func readSourceFiles(root string, dirs []string, keep func(rel string, d fs.DirEntry) bool) ([]sourceFile, error) {
	var out []sourceFile
	for _, dir := range dirs {
		base := filepath.Join(root, filepath.FromSlash(dir))
		err := filepath.WalkDir(base, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			rel, rerr := filepath.Rel(root, p)
			if rerr != nil {
				return rerr
			}
			rel = filepath.ToSlash(rel)
			if !keep(rel, d) {
				return nil
			}
			data, rerr := os.ReadFile(p) // #nosec G304 -- p is a file the walk just found under root
			if rerr != nil {
				return rerr
			}
			out = append(out, sourceFile{rel: rel, data: string(data)})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].rel < out[j].rel })
	return out, nil
}

func isGoSource(rel string, _ fs.DirEntry) bool {
	return strings.HasSuffix(rel, ".go") && !strings.HasSuffix(rel, "_test.go")
}

func isCSource(rel string, _ fs.DirEntry) bool {
	return strings.HasSuffix(rel, ".c")
}
