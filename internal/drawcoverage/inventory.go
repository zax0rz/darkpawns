package drawcoverage

import (
	"debug/dwarf"
	"debug/elf"
	"encoding/binary"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
)

// DrawSite is one call instruction in the C reference binary to a RNG draw
// primitive (number, dice, uniform, or prng_uniform).
type DrawSite struct {
	// Site is "src/file.c:line", or "src/file.c:line#N" when multiple call
	// instructions share one source line (e.g. src/weather.c:137).
	Site      string
	File      string
	Line      int
	Function  string
	Primitive string
	CallVaddr uint64
	RetVaddr  uint64
}

// Inventory is the complete set of RNG primitive call instructions extracted
// from the compiled C reference binary's ELF symbol table and DWARF line info.
type Inventory struct {
	// TopLevel are the player/game call sites whose calls are logged as
	// top-level draws by the C draw-log seam (excludes internal
	// primitive-to-primitive calls in src/utils.c).
	TopLevel []DrawSite
	// Internal are the primitive-to-primitive calls inside src/utils.c
	// (dice->number, number->uniform, uniform->prng_uniform).
	Internal []DrawSite
}

type funcRange struct {
	name  string
	start uint64
	end   uint64
}

type lineEntry struct {
	addr uint64
	file string
	line int
}

// ExtractInventory scans the .text section of oracleBin for every direct call
// instruction targeting number, dice, uniform, or prng_uniform, and resolves
// each call instruction to its enclosing function and DWARF source file:line.
func ExtractInventory(oracleBin string) (*Inventory, error) {
	f, err := elf.Open(oracleBin)
	if err != nil {
		return nil, fmt.Errorf("open ELF %s: %w", oracleBin, err)
	}
	defer func() { _ = f.Close() }()

	syms, err := f.Symbols()
	if err != nil {
		return nil, fmt.Errorf("read ELF symbols: %w", err)
	}

	targetByAddr := map[uint64]string{}
	var funcs []funcRange
	for _, s := range syms {
		if elf.ST_TYPE(s.Info) == elf.STT_FUNC && s.Size > 0 {
			funcs = append(funcs, funcRange{
				name:  s.Name,
				start: s.Value,
				end:   s.Value + s.Size,
			})
		}
		switch s.Name {
		case "number", "dice", "uniform", "prng_uniform":
			targetByAddr[s.Value] = s.Name
		}
	}
	if len(targetByAddr) < 4 {
		return nil, fmt.Errorf("expected 4 draw primitive symbols in %s, found %d", oracleBin, len(targetByAddr))
	}
	sort.Slice(funcs, func(i, j int) bool { return funcs[i].start < funcs[j].start })

	dw, err := f.DWARF()
	if err != nil {
		return nil, fmt.Errorf("read DWARF from %s: %w", oracleBin, err)
	}
	lines, err := loadDWARFLines(dw)
	if err != nil {
		return nil, fmt.Errorf("read DWARF line table: %w", err)
	}

	sec := f.Section(".text")
	if sec == nil {
		return nil, fmt.Errorf(".text section not found in %s", oracleBin)
	}
	data, err := sec.Data()
	if err != nil {
		return nil, fmt.Errorf("read .text: %w", err)
	}

	base := sec.Addr
	var all []DrawSite
	for i := 0; i+5 <= len(data); i++ {
		if data[i] != 0xe8 {
			continue
		}
		rel := int64(int32(binary.LittleEndian.Uint32(data[i+1 : i+5])))
		callVaddr := base + uint64(i)
		dest := uint64(int64(callVaddr+5) + rel)
		prim, ok := targetByAddr[dest]
		if !ok {
			continue
		}
		fn := lookupFunc(funcs, callVaddr)
		file, line := lookupLine(lines, callVaddr)
		all = append(all, DrawSite{
			File:      file,
			Line:      line,
			Function:  fn,
			Primitive: prim,
			CallVaddr: callVaddr,
			RetVaddr:  callVaddr + 5,
		})
	}

	inv := &Inventory{}
	for _, s := range all {
		if isInternalPrimitiveCall(s) {
			inv.Internal = append(inv.Internal, s)
		} else {
			inv.TopLevel = append(inv.TopLevel, s)
		}
	}

	assignSiteIDs(inv.TopLevel)
	assignSiteIDs(inv.Internal)
	return inv, nil
}

func isInternalPrimitiveCall(s DrawSite) bool {
	return (s.Function == "dice" && s.Primitive == "number") ||
		(s.Function == "number" && s.Primitive == "uniform") ||
		(s.Function == "uniform" && s.Primitive == "prng_uniform")
}

func assignSiteIDs(sites []DrawSite) {
	sort.Slice(sites, func(i, j int) bool {
		if sites[i].File != sites[j].File {
			return sites[i].File < sites[j].File
		}
		if sites[i].Line != sites[j].Line {
			return sites[i].Line < sites[j].Line
		}
		return sites[i].CallVaddr < sites[j].CallVaddr
	})
	counts := map[string]int{}
	for _, s := range sites {
		key := fmt.Sprintf("%s:%d", s.File, s.Line)
		counts[key]++
	}
	seen := map[string]int{}
	for i := range sites {
		key := fmt.Sprintf("%s:%d", sites[i].File, sites[i].Line)
		if counts[key] > 1 {
			seen[key]++
			sites[i].Site = fmt.Sprintf("%s#%d", key, seen[key])
		} else {
			sites[i].Site = key
		}
	}
}

func lookupFunc(funcs []funcRange, addr uint64) string {
	idx := sort.Search(len(funcs), func(i int) bool { return funcs[i].end > addr })
	if idx < len(funcs) && addr >= funcs[idx].start && addr < funcs[idx].end {
		return funcs[idx].name
	}
	return "<unknown>"
}

func loadDWARFLines(dw *dwarf.Data) ([]lineEntry, error) {
	var entries []lineEntry
	r := dw.Reader()
	for {
		e, err := r.Next()
		if err != nil {
			return nil, err
		}
		if e == nil {
			break
		}
		if e.Tag != dwarf.TagCompileUnit {
			r.SkipChildren()
			continue
		}
		lr, err := dw.LineReader(e)
		if err != nil {
			return nil, err
		}
		if lr == nil {
			r.SkipChildren()
			continue
		}
		var le dwarf.LineEntry
		for {
			err := lr.Next(&le)
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, err
			}
			if le.EndSequence || le.File == nil {
				continue
			}
			entries = append(entries, lineEntry{
				addr: le.Address,
				file: normalizeCPath(le.File.Name),
				line: le.Line,
			})
		}
		r.SkipChildren()
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].addr < entries[j].addr })
	return entries, nil
}

func lookupLine(lines []lineEntry, addr uint64) (string, int) {
	// Find the last entry with entry.addr <= addr.
	idx := sort.Search(len(lines), func(i int) bool { return lines[i].addr > addr }) - 1
	if idx >= 0 {
		return lines[idx].file, lines[idx].line
	}
	return "<unknown>", 0
}

func normalizeCPath(p string) string {
	p = filepath.ToSlash(filepath.Clean(p))
	for strings.HasPrefix(p, "./") {
		p = strings.TrimPrefix(p, "./")
	}
	if _, after, ok := strings.Cut(p, "/src/"); ok {
		return "src/" + after
	}
	for strings.HasPrefix(p, "src/src/") {
		p = strings.TrimPrefix(p, "src/")
	}
	return p
}
