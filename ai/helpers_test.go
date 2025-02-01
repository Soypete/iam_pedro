package ai

import "testing"

func Test_CleanResponse(t *testing.T) {
	tests := []struct {
		name string
		resp string
		want string
	}{
		{
			name: "Test 1",
			resp: "Hello\nWorld",
			want: "Hello World",
		},
		{
			name: "Test 2",
			resp: "<|im_start|> \nTtocsNeb: hi",
			want: "TtocsNeb: hi",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CleanResponse(tt.resp); got != tt.want {
				t.Errorf("CleanResponse() = %v, want %v", got, tt.want)
			}
		})
	}
}
