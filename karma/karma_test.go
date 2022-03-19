package karma

import (
	"reflect"
	"testing"
)

func TestBump(t *testing.T) {
	tests := []struct {
		name string
		curr map[string]int64
		incr map[string]int64
		want map[string]int64
	}{
		{
			name: "simple increment",
			curr: map[string]int64{"ogre": 0},
			incr: map[string]int64{"ogre": 1},
			want: map[string]int64{"ogre": 1},
		},
		{
			name: "simple decrement",
			curr: map[string]int64{"wolf": -1},
			incr: map[string]int64{"wolf": -1},
			want: map[string]int64{"wolf": -2},
		},
		{
			name: "net zero is ignored",
			curr: map[string]int64{"dune": 4},
			incr: map[string]int64{"dune": 0},
			want: map[string]int64{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Bump(tt.curr, tt.incr)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
