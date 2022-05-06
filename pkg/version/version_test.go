package version

import "testing"

func TestCurrent(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{name: version, want: version},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Current(); got != tt.want {
				t.Errorf("Current() = %v, want %v", got, tt.want)
			}
		})
	}
}
