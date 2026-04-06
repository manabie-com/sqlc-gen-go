package golang

import (
"testing"

"github.com/sqlc-dev/plugin-sdk-go/metadata"
"github.com/vtuanjs/sqlc-gen-go/internal/opts"
)

func baseImporter(sqlPkg string) *importer {
limit := int32(1)
initMap := map[string]struct{}{"id": {}}
return &importer{
Options: &opts.Options{
SqlPackage:          sqlPkg,
InitialismsMap:      initMap,
Rename:              map[string]string{},
QueryParameterLimit: &limit,
},
Queries: []Query{},
Enums:   []Enum{},
Structs: []Struct{},
}
}

func TestImportSpecString_WithID(t *testing.T) {
s := ImportSpec{ID: "pgx", Path: "github.com/jackc/pgx/v5"}
got := s.String()
want := `pgx "github.com/jackc/pgx/v5"`
if got != want {
t.Errorf("String() = %q, want %q", got, want)
}
}

func TestImportSpecString_NoID(t *testing.T) {
s := ImportSpec{Path: "context"}
got := s.String()
want := `"context"`
if got != want {
t.Errorf("String() = %q, want %q", got, want)
}
}

func TestMergeImports_Single(t *testing.T) {
fi := fileImports{
Std: []ImportSpec{{Path: "context"}},
Dep: []ImportSpec{{Path: "github.com/jackc/pgx/v5"}},
}
out := mergeImports(fi)
if len(out) != 2 {
t.Fatalf("expected 2 groups, got %d", len(out))
}
if len(out[0]) != 1 || out[0][0].Path != "context" {
t.Errorf("std group = %v", out[0])
}
}

func TestMergeImports_Multiple_Deduplicates(t *testing.T) {
fi1 := fileImports{
Std: []ImportSpec{{Path: "context"}, {Path: "fmt"}},
Dep: []ImportSpec{{Path: "github.com/foo/bar"}},
}
fi2 := fileImports{
Std: []ImportSpec{{Path: "context"}},
Dep: []ImportSpec{{Path: "github.com/baz/qux"}},
}
out := mergeImports(fi1, fi2)
if len(out[0]) != 2 {
t.Errorf("expected 2 std imports after dedup, got %d: %v", len(out[0]), out[0])
}
if len(out[1]) != 2 {
t.Errorf("expected 2 dep imports, got %d: %v", len(out[1]), out[1])
}
}

func TestImports_DbFile(t *testing.T) {
imp := baseImporter("database/sql")
out := imp.Imports("db.go")
if len(out) == 0 {
t.Error("expected non-empty imports for db.go")
}
}

func TestImports_ModelsFile(t *testing.T) {
imp := baseImporter("database/sql")
out := imp.Imports("models.go")
if len(out) == 0 {
t.Error("expected non-empty imports for models.go")
}
}

func TestImports_QuerierFile(t *testing.T) {
imp := baseImporter("database/sql")
out := imp.Imports("querier.go")
if len(out) == 0 {
t.Error("expected non-empty imports for querier.go")
}
}

func TestImports_CopyfromFile(t *testing.T) {
imp := baseImporter("database/sql")
imp.Queries = []Query{{
Cmd:        metadata.CmdCopyFrom,
SourceName: "users.sql",
Arg:        QueryValue{Typ: "string"},
}}
out := imp.Imports("copyfrom.go")
if len(out) == 0 {
t.Error("expected non-empty imports for copyfrom.go")
}
}

func TestImports_BatchFile(t *testing.T) {
imp := baseImporter("pgx/v5")
imp.Queries = []Query{{
Cmd:        metadata.CmdBatchExec,
SourceName: "users.sql",
}}
out := imp.Imports("batch.go")
if len(out) == 0 {
t.Error("expected non-empty imports for batch.go")
}
}

func TestImports_DynfilterFile(t *testing.T) {
imp := baseImporter("database/sql")
out := imp.Imports("dynfilter.go")
if len(out) != 2 {
t.Errorf("expected 2 groups for dynfilter.go, got %d", len(out))
}
if len(out[0]) != 0 || len(out[1]) != 0 {
t.Error("expected empty imports for dynfilter.go")
}
}

func TestImports_DefaultQuery(t *testing.T) {
imp := baseImporter("database/sql")
imp.Queries = []Query{{
Cmd:        metadata.CmdOne,
SourceName: "users.sql",
}}
out := imp.Imports("users.sql")
if len(out) == 0 {
t.Error("expected non-empty imports for query file")
}
}

func TestImports_CustomDbFileName(t *testing.T) {
imp := baseImporter("database/sql")
imp.Options.OutputDbFileName = "custom_db.go"
out := imp.Imports("custom_db.go")
if len(out) == 0 {
t.Error("expected non-empty imports for custom db file")
}
}

func TestDbImports_PGXv4(t *testing.T) {
imp := baseImporter("pgx/v4")
out := imp.Imports("db.go")
if len(out[1]) == 0 {
t.Error("expected pgx/v4 dep imports")
}
}

func TestDbImports_PGXv5(t *testing.T) {
imp := baseImporter("pgx/v5")
out := imp.Imports("db.go")
if len(out[1]) == 0 {
t.Error("expected pgx/v5 dep imports")
}
}

func TestInterfaceImports_Basic(t *testing.T) {
imp := baseImporter("database/sql")
imp.Queries = []Query{
{
Cmd: metadata.CmdOne,
Ret: QueryValue{Typ: "string"},
Arg: QueryValue{},
},
}
out := imp.interfaceImports()
found := false
for _, s := range out.Std {
if s.Path == "context" {
found = true
}
}
if !found {
t.Error("expected 'context' in interface imports")
}
}

func TestCopyfromImports_Basic(t *testing.T) {
imp := baseImporter("database/sql")
imp.Queries = []Query{{Cmd: metadata.CmdCopyFrom, Arg: QueryValue{Typ: "string"}}}
out := imp.copyfromImports()
found := false
for _, s := range out.Std {
if s.Path == "context" {
found = true
}
}
if !found {
t.Error("expected context in copyfrom imports")
}
}

func TestCopyfromImports_MySQLDriver(t *testing.T) {
imp := baseImporter("database/sql")
imp.Options.SqlDriver = opts.SQLDriverGoSQLDriverMySQL
imp.Queries = []Query{{Cmd: metadata.CmdCopyFrom, Arg: QueryValue{Typ: "string"}}}
out := imp.copyfromImports()
foundIO := false
for _, s := range out.Std {
if s.Path == "io" {
foundIO = true
}
}
if !foundIO {
t.Error("expected 'io' in MySQL copyfrom imports")
}
}

func TestBatchImports_PGXv5(t *testing.T) {
imp := baseImporter("pgx/v5")
imp.Queries = []Query{{Cmd: metadata.CmdBatchExec}}
out := imp.batchImports()
foundPgx := false
for _, s := range out.Dep {
if s.Path == "github.com/jackc/pgx/v5" {
foundPgx = true
}
}
if !foundPgx {
t.Error("expected pgx/v5 in batch imports")
}
}

func TestBatchImports_PGXv4(t *testing.T) {
imp := baseImporter("pgx/v4")
imp.Queries = []Query{{Cmd: metadata.CmdBatchExec}}
out := imp.batchImports()
foundPgx := false
for _, s := range out.Dep {
if s.Path == "github.com/jackc/pgx/v4" {
foundPgx = true
}
}
if !foundPgx {
t.Error("expected pgx/v4 in batch imports")
}
}
