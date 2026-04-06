package inflection

import "testing"

func TestSingular(t *testing.T) {
	cases := []struct {
		params SingularParams
		want   string
	}{
		{SingularParams{Name: "users"}, "user"},
		{SingularParams{Name: "orders"}, "order"},
		{SingularParams{Name: "campus"}, "campus"},
		{SingularParams{Name: "CAMPUS"}, "CAMPUS"},
		{SingularParams{Name: "meta"}, "meta"},
		{SingularParams{Name: "Meta"}, "Meta"},
		{SingularParams{Name: "calories"}, "calorie"},
		{SingularParams{Name: "waves"}, "wave"},
		{SingularParams{Name: "metadata"}, "metadata"},
		// exclusion takes precedence
		{SingularParams{Name: "sheep", Exclusions: []string{"sheep"}}, "sheep"},
		{SingularParams{Name: "Sheep", Exclusions: []string{"sheep"}}, "Sheep"},
	}
	for _, tc := range cases {
		t.Run(tc.params.Name, func(t *testing.T) {
			got := Singular(tc.params)
			if got != tc.want {
				t.Errorf("Singular(%q) = %q, want %q", tc.params.Name, got, tc.want)
			}
		})
	}
}
