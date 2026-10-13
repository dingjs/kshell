//go:build darwin

package desktop

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestComputePathAdditions(t *testing.T) {
	tmp := t.TempDir()
	mkdir := func(rel string) string {
		p := filepath.Join(tmp, rel)
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
		return p
	}
	bin := mkdir("bin")
	localBin := mkdir(".local/bin")

	cases := []struct {
		name string
		cur  string
		dirs []string
		want []string
	}{
		{
			name: "补缺失且存在的目录",
			cur:  "/usr/bin:/bin",
			dirs: []string{bin, localBin, "/no/such/dir"},
			want: []string{bin, localBin},
		},
		{
			name: "已存在的跳过不重复",
			cur:  "/usr/bin:" + bin,
			dirs: []string{bin, localBin},
			want: []string{localBin},
		},
		{
			name: "空 PATH",
			cur:  "",
			dirs: []string{bin, localBin},
			want: []string{bin, localBin},
		},
		{
			name: "展开 ~ 前缀",
			cur:  "/usr/bin:/bin",
			dirs: []string{"~/bin", "~/.local/bin", "/no/such/dir"},
			want: []string{filepath.Join(tmp, "bin"), filepath.Join(tmp, ".local", "bin")},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := computePathAdditions(c.cur, tmp, c.dirs)
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("got %v, want %v", got, c.want)
			}
		})
	}
}