package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func buildConfigDetails(source dict) ConfigDetails {
	return ConfigDetails{
		WorkingDir: ".",
		ConfigFiles: []ConfigFile{
			ConfigFile{Filename: "filename.yml", Config: source},
		},
		Environment: nil,
	}
}

func TestParseYAML(t *testing.T) {
	source := `
version: "2.1"
services:
  foo:
    image: busybox
  bar:
    image: busybox
    environment:
      - FOO=1
`

	configFile, err := ParseYAML([]byte(source), "filename.yml")
	if err != nil {
		t.Fatal(err)
	}

	expected := dict{
		"version": "2.1",
		"services": dict{
			"foo": dict{
				"image": "busybox",
			},
			"bar": dict{
				"image":       "busybox",
				"environment": []interface{}{"FOO=1"},
			},
		},
	}

	assert.Equal(t, expected, configFile.Config)
}

func TestLoad(t *testing.T) {
	source := dict{
		"version": "2.1",
		"services": dict{
			"foo": dict{
				"image": "busybox",
			},
			"bar": dict{
				"image":       "busybox",
				"environment": []string{"FOO=1"},
			},
		},
	}

	expected := Config{
		Services: []ServiceConfig{
			ServiceConfig{
				Name:        "foo",
				Image:       "busybox",
				Environment: nil,
			},
			ServiceConfig{
				Name:        "bar",
				Image:       "busybox",
				Environment: map[string]string{"FOO": "1"},
			},
		},
	}

	actual, err := Load(buildConfigDetails(source))
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, expected, *actual)
}
