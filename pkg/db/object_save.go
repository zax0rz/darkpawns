package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// ObjectSave is the independent C crash/rent-file identity and snapshot.
// The existing inventory/equipment JSON formats are unchanged.
type ObjectSave struct {
	Identity string
	Kind     int
	Objects  []game.SaveItemData
}

// ObjectSaveIdentity ports src/utils.c:527-586. The name is a logical identity
// in SQLite; it is not a claim that the port writes a filesystem sidecar.
func ObjectSaveIdentity(name string) string {
	name = strings.ToLower(name)
	if name == "" {
		return ""
	}
	group := "ZZZ"
	switch c := name[0]; {
	case c >= 'a' && c <= 'e':
		group = "A-E"
	case c >= 'f' && c <= 'j':
		group = "F-J"
	case c >= 'k' && c <= 'o':
		group = "K-O"
	case c >= 'p' && c <= 't':
		group = "P-T"
	case c >= 'u' && c <= 'z':
		group = "U-Z"
	}
	return "plrobjs/" + group + "/" + name + ".objs"
}

// SaveObjectSnapshot writes only at a real C object-save boundary. Unknown
// legacy histories stay absent until their next object save (Zach approval).
func SaveObjectSnapshot(store GameStore, name string, kind int, inventory, equipment []byte) error {
	objects, err := objectSaveRecords(inventory, equipment)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(objects)
	if err != nil {
		return err
	}
	_, err = store.Exec(`INSERT INTO object_saves(identity,kind,objects) VALUES(?,?,?) ON CONFLICT(identity) DO UPDATE SET kind=excluded.kind,objects=excluded.objects`, ObjectSaveIdentity(name), kind, raw)
	return err
}

func (d *DB) GetObjectSave(name string) (*ObjectSave, error) {
	s := &ObjectSave{}
	var raw []byte
	err := d.queryRow("SELECT identity,kind,objects FROM object_saves WHERE identity=?", ObjectSaveIdentity(name)).Scan(&s.Identity, &s.Kind, &raw)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(raw, &s.Objects); err != nil {
		return nil, err
	}
	return s, nil
}

// ObjectSaveLoaded is Crash_load's header rewrite (src/objsave.c:659-663).
// UPDATE preserves absence: entering alone never creates a crash file.
func ObjectSaveLoaded(store GameStore, name string) error {
	_, err := store.Exec("UPDATE object_saves SET kind=1 WHERE identity=?", ObjectSaveIdentity(name))
	return err
}

// objectSaveRecords projects parent-first stored trees to Crash_save's
// sibling/child/object order and signed depth (src/objsave.c:673-680).
func objectSaveRecords(inventory, equipment []byte) ([]game.SaveItemData, error) {
	out := make([]game.SaveItemData, 0)
	for index, raw := range [][]byte{equipment, inventory} {
		var items []game.SaveItemData
		if len(raw) == 0 || string(raw) == "{}" {
			continue
		}
		if err := json.Unmarshal(raw, &items); err != nil {
			return nil, fmt.Errorf("object-save topology: %w", err)
		}
		children := make(map[int][]int)
		var roots []int
		for i, item := range items {
			if item.ContainerIndex < 0 || item.ContainerIndex > i {
				return nil, fmt.Errorf("invalid saved parent at %d", i)
			}
			if item.ContainerIndex == 0 {
				roots = append(roots, i)
			} else {
				children[item.ContainerIndex-1] = append(children[item.ContainerIndex-1], i)
			}
		}
		if index == 0 {
			sort.SliceStable(roots, func(i, j int) bool { return items[roots[i]].Locate < items[roots[j]].Locate })
		} else {
			for i, j := 0, len(roots)-1; i < j; i, j = i+1, j-1 {
				roots[i], roots[j] = roots[j], roots[i]
			}
		}
		var visit func(int, int)
		visit = func(i, locate int) {
			descendants := children[i]
			depth := min(0, locate) - 1
			for j := len(descendants) - 1; j >= 0; j-- {
				visit(descendants[j], depth)
			}
			item := items[i]
			item.Locate = locate
			// C flat records encode ancestry only through signed locate.
			item.ContainerIndex, item.ContainerVNum = 0, 0
			count := max(1, item.Count)
			item.Count = 1
			for range count {
				out = append(out, item)
			}
		}
		for _, root := range roots {
			visit(root, items[root].Locate)
		}
	}
	return out, nil
}

// DeleteObjectSave is Crash_delete_file, keeping the logical identity separate
// from character deletion or name reuse (src/interpreter.c:2339).
func DeleteObjectSave(store GameStore, name string) error {
	_, err := store.Exec("DELETE FROM object_saves WHERE identity=?", ObjectSaveIdentity(name))
	return err
}
