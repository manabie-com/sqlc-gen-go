package golang

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/sqlc-dev/plugin-sdk-go/plugin"
)

// ifAnnotationRe matches "-- :if @paramName" or "-- :if $paramName"
var ifAnnotationRe = regexp.MustCompile(`--\s*:if\s+[@$](\w+)\s*$`)

// DynFilterInfo holds the result of parsing :if annotations from a SQL query.
type DynFilterInfo struct {
	// AnnotatedSQL is the SQL with -- :if replaced by -- :dynif N markers.
	// N is the 0-based index into the full args slice passed to DynamicSQL.
	// For SQL params: N = paramNumber - 1  (matches $N position in args).
	// For flag-only params: N = numSQLParams + flagOffset.
	AnnotatedSQL string
	// ConditionalParamNumbers contains the $N numbers (1-based) of SQL params
	// that are conditional and should become pointer types.
	ConditionalParamNumbers []int
	// FlagParams are extra bool params that need to be added to the params struct
	// (params referenced in :if that are not actual SQL params).
	FlagParams []FlagParam
	// OrderedArgNames is the full ordered list of param names for the DynamicSQL
	// call, indexed by their :dynif N value.
	// Positions 0..numSQLParams-1 are SQL params (in $N order, all of them).
	// Positions numSQLParams.. are flag-only params (in appearance order).
	OrderedArgNames []string
}

// FlagParam represents a flag-only bool parameter (used in ORDER BY :if).
type FlagParam struct {
	// Name is the original @name from the :if annotation.
	Name string
	// GoName is the CamelCase Go field name.
	GoName string
}

// ParseDynFilter parses -- :if @param annotations from SQL query text.
// params is the list of ALL SQL parameters (from sqlc).
//
// The :dynif N index assigned to each annotation equals paramNumber-1 for
// SQL params, so that the DynamicSQL runtime can directly use args[N] ↔ $N+1.
// Flag-only params (ORDER BY flags not in SQL) get indices starting at
// len(params), and are appended to the DynamicSQL args after the SQL params.
//
// Both inline and block syntax are supported:
//
//	AND b = $2 -- :if @b          (inline)
//	-- :if @b                      (block: applies to next line)
//	AND b = $2
func ParseDynFilter(sql string, params []*plugin.Parameter) (*DynFilterInfo, error) {
	// Build map: column name -> param number (1-based)
	paramByName := make(map[string]int32)
	for _, p := range params {
		if p.Column.Name != "" {
			paramByName[p.Column.Name] = p.Number
		}
	}

	lines := strings.Split(sql, "\n")

	// First pass: collect all :if annotations to find which params are
	// conditional and which are flag-only.
	type refEntry struct {
		name        string
		isFlagOnly  bool
		paramNumber int32 // only valid if !isFlagOnly; equals the $N number
	}
	seenName := make(map[string]bool)
	var refs []refEntry

	for _, line := range lines {
		m := ifAnnotationRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		name := m[1]
		if seenName[name] {
			continue
		}
		seenName[name] = true
		if paramNum, ok := paramByName[name]; ok {
			refs = append(refs, refEntry{name: name, isFlagOnly: false, paramNumber: paramNum})
		} else {
			refs = append(refs, refEntry{name: name, isFlagOnly: true})
		}
	}

	if len(refs) == 0 {
		return nil, nil
	}

	// Build name -> :dynif index mapping.
	// SQL params: index = paramNumber - 1  (0-based, matches $N position in args)
	// Flag params: index = len(params) + flagOffset  (appended after SQL params)
	argIndexByName := make(map[string]int)
	conditionalParamNums := make(map[int32]bool)
	var flagParams []FlagParam
	flagOffset := 0
	for _, r := range refs {
		if !r.isFlagOnly {
			argIndexByName[r.name] = int(r.paramNumber) - 1
			conditionalParamNums[r.paramNumber] = true
		} else {
			argIndexByName[r.name] = len(params) + flagOffset
			flagParams = append(flagParams, FlagParam{
				Name:   r.name,
				GoName: structName(r.name),
			})
			flagOffset++
		}
	}

	// Second pass: rewrite the SQL, replacing -- :if @name with -- :dynif N.
	var newLines []string
	for _, line := range lines {
		m := ifAnnotationRe.FindStringSubmatch(line)
		if m == nil {
			newLines = append(newLines, line)
			continue
		}
		name := m[1]
		idx, ok := argIndexByName[name]
		if !ok {
			return nil, fmt.Errorf("dynfilter: unknown param @%s", name)
		}

		annotationStart := ifAnnotationRe.FindStringIndex(line)
		prefix := strings.TrimSpace(line[:annotationStart[0]])
		if prefix == "" {
			// Standalone block annotation
			newLines = append(newLines, fmt.Sprintf("-- :if $%d", idx+1))
		} else {
			// Inline annotation
			newLine := strings.TrimRight(line[:annotationStart[0]], " \t") + fmt.Sprintf(" -- :if $%d", idx+1)
			newLines = append(newLines, newLine)
		}
	}

	annotatedSQL := strings.Join(newLines, "\n")

	// Build ConditionalParamNumbers list
	var condNums []int
	for num := range conditionalParamNums {
		condNums = append(condNums, int(num))
	}
	sort.Ints(condNums)

	// Build OrderedArgNames: all SQL params in $N order (position 0..N-1),
	// then flag params in appearance order.
	// Collect param names sorted by number.
	type sqlParam struct {
		name   string
		number int32
	}
	var sqlParamsSorted []sqlParam
	for _, p := range params {
		if p.Column.Name != "" {
			sqlParamsSorted = append(sqlParamsSorted, sqlParam{name: p.Column.Name, number: p.Number})
		}
	}
	sort.Slice(sqlParamsSorted, func(i, j int) bool {
		return sqlParamsSorted[i].number < sqlParamsSorted[j].number
	})

	orderedArgNames := make([]string, len(params)+len(flagParams))
	for i, sp := range sqlParamsSorted {
		orderedArgNames[i] = sp.name
	}
	for i, fp := range flagParams {
		orderedArgNames[len(params)+i] = fp.Name
	}

	return &DynFilterInfo{
		AnnotatedSQL:            annotatedSQL,
		ConditionalParamNumbers: condNums,
		FlagParams:              flagParams,
		OrderedArgNames:         orderedArgNames,
	}, nil
}

// structName converts snake_case to CamelCase (reuses the same logic as StructName but simplified).
func structName(name string) string {
	parts := strings.Split(name, "_")
	var out string
	for _, p := range parts {
		if len(p) == 0 {
			continue
		}
		out += strings.ToUpper(p[:1]) + p[1:]
	}
	return out
}
