package helmengine

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestMergeInlineValues(t *testing.T) {
	tests := []struct {
		name   string
		base   map[string]interface{}
		inline map[string]interface{}
		want   map[string]interface{}
	}{
		{
			name:   "nil base",
			base:   nil,
			inline: map[string]interface{}{"a": 1},
			want:   map[string]interface{}{"a": 1},
		},
		{
			name: "nested maps merge instead of replacing siblings",
			base: map[string]interface{}{
				"image": map[string]interface{}{"repository": "nginx", "tag": "1.0"},
			},
			inline: map[string]interface{}{
				"image": map[string]interface{}{"tag": "2.0"},
			},
			want: map[string]interface{}{
				"image": map[string]interface{}{"repository": "nginx", "tag": "2.0"},
			},
		},
		{
			name:   "scalar replaces map",
			base:   map[string]interface{}{"a": map[string]interface{}{"b": 1}},
			inline: map[string]interface{}{"a": "flat"},
			want:   map[string]interface{}{"a": "flat"},
		},
		{
			name:   "map replaces scalar",
			base:   map[string]interface{}{"a": "flat"},
			inline: map[string]interface{}{"a": map[string]interface{}{"b": 1}},
			want:   map[string]interface{}{"a": map[string]interface{}{"b": 1}},
		},
		{
			name:   "null is kept so Helm can delete the key",
			base:   map[string]interface{}{"a": 1, "b": 2},
			inline: map[string]interface{}{"a": nil},
			want:   map[string]interface{}{"a": nil, "b": 2},
		},
		{
			name:   "lists are replaced, not appended",
			base:   map[string]interface{}{"l": []interface{}{1, 2}},
			inline: map[string]interface{}{"l": []interface{}{3}},
			want:   map[string]interface{}{"l": []interface{}{3}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeInlineValues(tt.base, tt.inline)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestApplySetValues(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "cert.pem")
	if err := os.WriteFile(file, []byte("PEM DATA"), 0o600); err != nil {
		t.Fatal(err)
	}

	base := map[string]interface{}{
		"image": map[string]interface{}{"repository": "nginx", "tag": 1},
	}
	got, err := ApplySetValues(base, ValueSetOptions{
		SetString:  []string{"image.tag=007"},
		SetFile:    []string{"tls.cert=" + file},
		SetLiteral: []string{"literal=a,b=c"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	image := got["image"].(map[string]interface{})
	if image["tag"] != "007" {
		t.Errorf("set_string should keep the string form, got %#v", image["tag"])
	}
	if image["repository"] != "nginx" {
		t.Errorf("sibling keys must survive, got %#v", image)
	}
	if got["tls"].(map[string]interface{})["cert"] != "PEM DATA" {
		t.Errorf("set_file should read the file, got %#v", got["tls"])
	}
	if got["literal"] != "a,b=c" {
		t.Errorf("set_literal should not split, got %#v", got["literal"])
	}
}

func TestApplySetValues_Errors(t *testing.T) {
	tests := []struct {
		name string
		opts ValueSetOptions
		want string
	}{
		{"bad set_string", ValueSetOptions{SetString: []string{"noequals"}}, "set_string"},
		{"missing set_file", ValueSetOptions{SetFile: []string{"k=/does/not/exist"}}, "set_file"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ApplySetValues(nil, tt.opts)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q error, got %v", tt.want, err)
			}
		})
	}
}

func TestSetFilePaths(t *testing.T) {
	got := SetFilePaths([]string{"a=/tmp/x", "b.c=rel/y", "broken"})
	want := []string{"/tmp/x", "rel/y"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestResourceRefs(t *testing.T) {
	svc := &corev1.Service{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Service"},
		ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "prod"},
	}
	cm := &corev1.ConfigMap{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "ConfigMap"},
		ObjectMeta: metav1.ObjectMeta{Name: "cfg", Namespace: "prod"},
	}
	got := ResourceRefs(map[string][]runtime.Object{
		"v1/Service":   {svc},
		"v1/ConfigMap": {cm},
	})
	want := []ResourceRef{
		{APIVersion: "v1", Kind: "ConfigMap", Name: "cfg", Namespace: "prod"},
		{APIVersion: "v1", Kind: "Service", Name: "web", Namespace: "prod"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
	if ResourceRefs(nil) != nil {
		t.Error("nil resources should give a nil slice")
	}
}

func TestExpandSubchartPaths(t *testing.T) {
	root := t.TempDir()
	parent := filepath.Join(root, "parent")
	sub := filepath.Join(parent, "charts", "sub")
	if err := os.MkdirAll(sub, 0o750); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{filepath.Join(parent, "Chart.yaml"), filepath.Join(sub, "Chart.yaml"), filepath.Join(parent, "charts", "packed-0.1.0.tgz")} {
		if err := os.WriteFile(f, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	got := ExpandSubchartPaths([]string{parent})
	want := []string{parent, filepath.Join(parent, "charts", "packed-0.1.0.tgz"), sub}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	if got := ExpandSubchartPaths([]string{filepath.Join(root, "missing")}); len(got) != 1 {
		t.Errorf("a path without charts/ should be returned unchanged, got %v", got)
	}
}

func TestNewClientTLSConfig(t *testing.T) {
	cfg, err := NewClientTLSConfig("", "", "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.InsecureSkipVerify || cfg.RootCAs != nil || len(cfg.Certificates) != 0 {
		t.Errorf("empty inputs should give a default config, got %+v", cfg)
	}

	cfg, err = NewClientTLSConfig("", "", "", true)
	if err != nil || !cfg.InsecureSkipVerify {
		t.Errorf("insecure should be honoured, got %+v, %v", cfg, err)
	}

	dir := t.TempDir()
	notPEM := filepath.Join(dir, "ca.pem")
	if err := os.WriteFile(notPEM, []byte("not a certificate"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewClientTLSConfig("", "", notPEM, false); err == nil {
		t.Error("a CA file with no certificates should be rejected")
	}
	if _, err := NewClientTLSConfig("", "", filepath.Join(dir, "missing.pem"), false); err == nil {
		t.Error("a missing CA file should be rejected")
	}
	if _, err := NewClientTLSConfig(notPEM, notPEM, "", false); err == nil {
		t.Error("an invalid key pair should be rejected")
	}
}
