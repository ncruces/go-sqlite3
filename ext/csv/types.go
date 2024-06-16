package csv

import (
	"context"
	_ "embed"
	"strings"

	"github.com/tetratelabs/wazero"
)

type affinity byte

const (
	blob    affinity = 0
	text    affinity = 1
	numeric affinity = 2
	integer affinity = 3
	real    affinity = 4
)

//go:embed parser/sql3parse_table.wasm
var binary []byte

func getColumnAffinities(schema string) []affinity {
	ctx := context.Background()

	runtime := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigInterpreter())
	defer runtime.Close(ctx)

	mod, err := runtime.Instantiate(ctx, binary)
	if err != nil {
		return nil
	}

	if buf, ok := mod.Memory().Read(4, uint32(len(schema))); ok {
		copy(buf, schema)
	} else {
		return nil
	}

	r, err := mod.ExportedFunction("sql3parse_table").Call(ctx, 4, uint64(len(schema)), 0)
	if err != nil || r[0] == 0 {
		return nil
	}
	table := r[0]

	r, err = mod.ExportedFunction("sql3table_num_columns").Call(ctx, table)
	if err != nil {
		return nil
	}
	types := make([]affinity, r[0])

	for i := range types {
		r, err = mod.ExportedFunction("sql3table_get_column").Call(ctx, table, uint64(i))
		if err != nil || r[0] == 0 {
			break
		}
		r, err = mod.ExportedFunction("sql3column_type").Call(ctx, r[0])
		if err != nil || r[0] == 0 {
			continue
		}

		str, ok := mod.Memory().ReadUint32Le(uint32(r[0]) + 0)
		if !ok {
			break
		}
		len, ok := mod.Memory().ReadUint32Le(uint32(r[0]) + 4)
		if !ok {
			break
		}
		name, ok := mod.Memory().Read(str, len)
		if !ok {
			break
		}
		types[i] = getAffinity(string(name))
	}
	return types
}

func getAffinity(declType string) affinity {
	// https://sqlite.org/datatype3.html#determination_of_column_affinity
	if declType == "" {
		return blob
	}
	name := strings.ToUpper(declType)
	if strings.Contains(name, "INT") {
		return integer
	}
	if strings.Contains(name, "CHAR") || strings.Contains(name, "CLOB") || strings.Contains(name, "TEXT") {
		return text
	}
	if strings.Contains(name, "BLOB") {
		return blob
	}
	if strings.Contains(name, "REAL") || strings.Contains(name, "FLOA") || strings.Contains(name, "DOUB") {
		return real
	}
	return numeric
}
