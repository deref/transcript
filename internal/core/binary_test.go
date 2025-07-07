package core

import "testing"

func TestIsBinary(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{
			name: "empty data",
			data: []byte{},
			want: false,
		},
		{
			name: "normal text",
			data: []byte("hello world"),
			want: false,
		},
		{
			name: "text with newlines",
			data: []byte("hello\nworld\n"),
			want: false,
		},
		{
			name: "text with null byte",
			data: []byte("hello\x00world"),
			want: true,
		},
		{
			name: "text with multiple null bytes",
			data: []byte("\x00\x00\x00"),
			want: true,
		},
		{
			name: "text with high unprintable ratio",
			data: []byte("a\x01\x02\x03\x04\x05\x06\x07\x08\x09"), // 9 unprintable out of 10 = 90%
			want: true,
		},
		{
			name: "text with low unprintable ratio",
			data: []byte("hello world\x01"), // 1 unprintable out of 12 = 8.3%
			want: false,
		},
		{
			name: "text exactly at 10% threshold",
			data: []byte("abcdefghi\x01"), // 1 unprintable out of 10 = 10%
			want: false,
		},
		{
			name: "text just over 10% threshold",
			data: []byte("abcdefgh\x01\x02"), // 2 unprintable out of 10 = 20%
			want: true,
		},
		{
			name: "single byte printable",
			data: []byte("a"),
			want: false,
		},
		{
			name: "single byte unprintable",
			data: []byte{0x01},
			want: true,
		},
		{
			name: "unicode text",
			data: []byte("hello 世界"),
			want: false,
		},
		{
			name: "invalid utf8",
			data: []byte{0x80, 0x81, 0x82},
			want: true,
		},
		{
			name: "tab and space characters",
			data: []byte("hello\tworld\n"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isBinary(tt.data)
			if got != tt.want {
				t.Errorf("isBinary(%q) = %v, want %v", tt.data, got, tt.want)
			}
		})
	}
}
