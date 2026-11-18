# C-to-Go function citation ratchet

`make function-index` regenerates `index.tsv` from Go AST declarations and C
definitions. `make function-index-check` reads the Go source comments directly;
it does not trust a previously generated index.

`baseline.tsv` records the reviewed C-to-Go citation pairs present when the
ratchet was introduced. `primary` is the selected Go association for a C
function with one mapping; `split` records an accepted additional association
for a helper, caller, wrapper, or separated implementation slice; `duplicate`
marks one of the exact Go body copies listed in `duplicates.tsv`. These labels
describe reviewed citation edges, not oracle-proven behavior equivalence.

When adding another Go citation for a C function that already has a Go port,
verify the C and Go call paths (R5c) and add the pair to the baseline with its
reviewed class. The check rejects an unbaselined pair even if the old citation
is removed in the same change; for a C function without any baseline port, it
rejects multiple simultaneous Go associations.

The duplicate list is shrink-only. To retire one, remove its C citation from
the duplicate Go function and remove the row from `duplicates.tsv` in the same
change. The baseline row remains as history; if that citation later returns,
the checker fails until Zach explicitly reviews the change. Do not add a
duplicate row without a corresponding `duplicate` baseline classification.

The ratchet's CI control temporarily adds a fake `C: is_number()` function to
`pkg/game`, confirms that it fails as an unbaselined association, removes the
file, and confirms that the real tree is clean.
