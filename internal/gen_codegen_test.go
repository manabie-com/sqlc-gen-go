package golang

import (
	"testing"

	"github.com/sqlc-dev/plugin-sdk-go/metadata"
	"github.com/vtuanjs/sqlc-gen-go/internal/opts"
)

// ---- codegenDbarg -----------------------------------------------------------

func TestCodegenDbarg(t *testing.T) {
	ctx := &tmplCtx{EmitMethodsWithDBArgument: false}
	if got := ctx.codegenDbarg(); got != "" {
		t.Errorf("codegenDbarg (false) = %q, want empty", got)
	}
	ctx.EmitMethodsWithDBArgument = true
	if got := ctx.codegenDbarg(); got != "db DBTX, " {
		t.Errorf("codegenDbarg (true) = %q, want \"db DBTX, \"", got)
	}
}

// ---- codegenEmitPreparedQueries ---------------------------------------------

func TestCodegenEmitPreparedQueries(t *testing.T) {
	ctx := &tmplCtx{EmitPreparedQueries: true}
	if !ctx.codegenEmitPreparedQueries() {
		t.Error("expected true")
	}
	ctx.EmitPreparedQueries = false
	if ctx.codegenEmitPreparedQueries() {
		t.Error("expected false")
	}
}

// ---- codegenQueryMethod -----------------------------------------------------

func TestCodegenQueryMethod(t *testing.T) {
	cases := []struct {
		dbArg    bool
		prepared bool
		cmd      string
		want     string
	}{
		{false, false, ":one", "q.db.QueryRowContext"},
		{false, false, ":many", "q.db.QueryContext"},
		{false, false, ":exec", "q.db.ExecContext"},
		{true, false, ":one", "db.QueryRowContext"},
		{true, false, ":many", "db.QueryContext"},
		{true, false, ":exec", "db.ExecContext"},
		{false, true, ":one", "q.queryRow"},
		{false, true, ":many", "q.query"},
		{false, true, ":exec", "q.exec"},
	}
	for _, tc := range cases {
		ctx := &tmplCtx{EmitMethodsWithDBArgument: tc.dbArg, EmitPreparedQueries: tc.prepared}
		q := Query{Cmd: tc.cmd}
		got := ctx.codegenQueryMethod(q)
		if got != tc.want {
			t.Errorf("codegenQueryMethod(dbArg=%v, prepared=%v, cmd=%q) = %q, want %q",
				tc.dbArg, tc.prepared, tc.cmd, got, tc.want)
		}
	}
}

// ---- codegenQueryRetval -----------------------------------------------------

func TestCodegenQueryRetval(t *testing.T) {
	ctx := &tmplCtx{}
	cases := []struct {
		cmd     string
		want    string
		wantErr bool
	}{
		{":one", "row :=", false},
		{":many", "rows, err :=", false},
		{":exec", "_, err :=", false},
		{":execrows", "result, err :=", false},
		{":execlastid", "result, err :=", false},
		{":execresult", "return", false},
		{":unknown", "", true},
	}
	for _, tc := range cases {
		q := Query{Cmd: tc.cmd}
		got, err := ctx.codegenQueryRetval(q)
		if tc.wantErr {
			if err == nil {
				t.Errorf("codegenQueryRetval(%q): expected error", tc.cmd)
			}
			continue
		}
		if err != nil {
			t.Errorf("codegenQueryRetval(%q): unexpected error: %v", tc.cmd, err)
		}
		if got != tc.want {
			t.Errorf("codegenQueryRetval(%q) = %q, want %q", tc.cmd, got, tc.want)
		}
	}
}

// ---- codegenTracingCode -----------------------------------------------------

func TestCodegenTracingCode_Nil(t *testing.T) {
	ctx := &tmplCtx{}
	got, err := ctx.codegenTracingCode("DoSomething")
	if err != nil || got != "" {
		t.Errorf("expected empty, no error; got %q, %v", got, err)
	}
}

func TestCodegenTracingCode_WithCode(t *testing.T) {
	ctx := &tmplCtx{
		StructName: "Queries",
		EmitTracing: &opts.TracingOptions{
			Code: []string{`// trace {{.MethodName}} on {{.StructName}}`},
		},
	}
	got, err := ctx.codegenTracingCode("GetUser")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "// trace GetUser on Queries\n" {
		t.Errorf("codegenTracingCode = %q", got)
	}
}

func TestCodegenTracingCode_InvalidTemplate(t *testing.T) {
	ctx := &tmplCtx{
		EmitTracing: &opts.TracingOptions{
			Code: []string{"{{.Invalid"},
		},
	}
	_, err := ctx.codegenTracingCode("X")
	if err == nil {
		t.Error("expected error for invalid template")
	}
}

// ---- usesCopyFrom / usesDynFilter / usesBatch --------------------------------

func TestUsesCopyFrom(t *testing.T) {
	q := []Query{{Cmd: metadata.CmdCopyFrom}}
	if !usesCopyFrom(q) {
		t.Error("expected true for :copyfrom query")
	}
	if usesCopyFrom([]Query{{Cmd: metadata.CmdOne}}) {
		t.Error("expected false for non-copyfrom query")
	}
}

func TestUsesDynFilter(t *testing.T) {
	q := []Query{{HasDynFilter: true}}
	if !usesDynFilter(q) {
		t.Error("expected true for dynfilter query")
	}
	if usesDynFilter([]Query{{HasDynFilter: false}}) {
		t.Error("expected false")
	}
}

func TestUsesBatch(t *testing.T) {
	cmds := []string{metadata.CmdBatchExec, metadata.CmdBatchMany, metadata.CmdBatchOne}
	for _, cmd := range cmds {
		if !usesBatch([]Query{{Cmd: cmd}}) {
			t.Errorf("usesBatch should be true for %q", cmd)
		}
	}
	if usesBatch([]Query{{Cmd: metadata.CmdOne}}) {
		t.Error("expected false for :one")
	}
}

// ---- checkNoTimesForMySQLCopyFrom -------------------------------------------

func TestCheckNoTimesForMySQLCopyFrom_OK(t *testing.T) {
	q := []Query{
		{
			Cmd: metadata.CmdCopyFrom,
			Arg: QueryValue{
				Struct: &Struct{Fields: []Field{{Type: "string"}, {Type: "int32"}}},
			},
		},
	}
	if err := checkNoTimesForMySQLCopyFrom(q); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCheckNoTimesForMySQLCopyFrom_TimeFails(t *testing.T) {
	q := []Query{
		{
			Cmd: metadata.CmdCopyFrom,
			Arg: QueryValue{
				Struct: &Struct{Fields: []Field{{Type: "time.Time"}}},
			},
		},
	}
	if err := checkNoTimesForMySQLCopyFrom(q); err == nil {
		t.Error("expected error for time.Time field in CopyFrom")
	}
}

func TestCheckNoTimesForMySQLCopyFrom_SkipsNonCopyFrom(t *testing.T) {
	q := []Query{
		{
			Cmd: metadata.CmdOne,
			Arg: QueryValue{
				Struct: &Struct{Fields: []Field{{Type: "time.Time"}}},
			},
		},
	}
	if err := checkNoTimesForMySQLCopyFrom(q); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// ---- validate ---------------------------------------------------------------

func defaultValidateOpts() *opts.Options {
	return &opts.Options{InitialismsMap: map[string]struct{}{}}
}

func TestValidate_NoConflict(t *testing.T) {
	if err := validate(defaultValidateOpts(), nil, nil, nil); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidate_StructEnumConflict(t *testing.T) {
	enums := []Enum{{Name: "Status"}}
	structs := []Struct{{Name: "Status"}}
	if err := validate(defaultValidateOpts(), enums, structs, nil); err == nil {
		t.Error("expected error for struct/enum name conflict")
	}
}

func TestValidate_ExportedQueryEnumConflict(t *testing.T) {
	o := defaultValidateOpts()
	o.EmitExportedQueries = true
	enums := []Enum{{Name: "Status"}}
	queries := []Query{{ConstantName: "Status"}}
	if err := validate(o, enums, nil, queries); err == nil {
		t.Error("expected error for query constant/enum name conflict")
	}
}

func TestValidate_ExportedQueryStructConflict(t *testing.T) {
	o := defaultValidateOpts()
	o.EmitExportedQueries = true
	structs := []Struct{{Name: "GetUser"}}
	queries := []Query{{ConstantName: "GetUser"}}
	if err := validate(o, nil, structs, queries); err == nil {
		t.Error("expected error for query constant/struct name conflict")
	}
}

func TestValidate_ExportedQueryNoConflict(t *testing.T) {
	o := defaultValidateOpts()
	o.EmitExportedQueries = true
	if err := validate(o, nil, nil, []Query{{ConstantName: "GetUser"}}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
