# Entry-name design reproducer

Base `5a060e692`. Copy probe.go.txt to
pkg/session/entry_name_design_probe_test.go in a disposable base checkout and run:

```
go test ./pkg/session -run '^TestEntryNameDesignProbe$' -count=1
```

Expected exit 1: four assertion failures, each reaching confirm_name. Remove the
temporary test afterward. This demonstrates the real Go login branch; C
rejection follows the read source sites in the design note. No oracle execution,
passing fidelity proof, status change or production fix is claimed.
