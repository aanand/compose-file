package main

import (
	"sort"
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

var sampleYAML = []byte(`
version: "2.1"
services:
  foo:
    image: busybox
  bar:
    image: busybox
    environment:
      - FOO=1
`)

var sampleDict = dict{
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

var sampleConfig = Config{
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

func TestParseYAML(t *testing.T) {
	configFile, err := ParseYAML(sampleYAML, "filename.yml")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, sampleDict, configFile.Config)
}

func TestLoad(t *testing.T) {
	actual, err := Load(buildConfigDetails(sampleDict))
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, serviceSort(sampleConfig.Services), serviceSort(actual.Services))
}

func serviceSort(services []ServiceConfig) []ServiceConfig {
	sort.Sort(servicesByName(services))
	return services
}

type servicesByName []ServiceConfig

func (sbn servicesByName) Len() int           { return len(sbn) }
func (sbn servicesByName) Swap(i, j int)      { sbn[i], sbn[j] = sbn[j], sbn[i] }
func (sbn servicesByName) Less(i, j int) bool { return sbn[i].Name < sbn[j].Name }
