package loader

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aanand/compose-file/types"
)

func buildConfigDetails(source types.Dict) types.ConfigDetails {
	return types.ConfigDetails{
		WorkingDir: ".",
		ConfigFiles: []types.ConfigFile{
			types.ConfigFile{Filename: "filename.yml", Config: source},
		},
		Environment: nil,
	}
}

var sampleYAML = `
version: "2.1"
services:
  foo:
    image: busybox
  bar:
    image: busybox
    environment:
      - FOO=1
volumes:
  hello:
    driver: default
    driver_opts:
      beep: boop
networks:
  default:
    driver: bridge
    driver_opts:
      beep: boop
  with_ipam:
    ipam:
      driver: default
      config:
        - subnet: 172.28.0.0/16
`

var sampleDict = types.Dict{
	"version": "2.1",
	"services": types.Dict{
		"foo": types.Dict{
			"image": "busybox",
		},
		"bar": types.Dict{
			"image":       "busybox",
			"environment": []interface{}{"FOO=1"},
		},
	},
	"volumes": types.Dict{
		"hello": types.Dict{
			"driver": "default",
			"driver_opts": types.Dict{
				"beep": "boop",
			},
		},
	},
	"networks": types.Dict{
		"default": types.Dict{
			"driver": "bridge",
			"driver_opts": types.Dict{
				"beep": "boop",
			},
		},
		"with_ipam": types.Dict{
			"ipam": types.Dict{
				"driver": "default",
				"config": []interface{}{
					types.Dict{
						"subnet": "172.28.0.0/16",
					},
				},
			},
		},
	},
}

var sampleConfig = types.Config{
	Services: []types.ServiceConfig{
		types.ServiceConfig{
			Name:        "foo",
			Image:       "busybox",
			Environment: nil,
		},
		types.ServiceConfig{
			Name:        "bar",
			Image:       "busybox",
			Environment: map[string]string{"FOO": "1"},
		},
	},
	Networks: map[string]types.NetworkConfig{
		"default": types.NetworkConfig{
			Driver: "bridge",
			DriverOpts: map[string]string{
				"beep": "boop",
			},
		},
		"with_ipam": types.NetworkConfig{
			IPAM: types.IPAMConfig{
				Driver: "default",
				Config: []types.IPAMPool{
					types.IPAMPool{
						Subnet: "172.28.0.0/16",
					},
				},
			},
		},
	},
	Volumes: map[string]types.VolumeConfig{
		"hello": types.VolumeConfig{
			Driver: "default",
			DriverOpts: map[string]string{
				"beep": "boop",
			},
		},
	},
}

func TestParseYAML(t *testing.T) {
	configFile, err := ParseYAML([]byte(sampleYAML), "filename.yml")
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
	assert.Equal(t, sampleConfig.Networks, actual.Networks)
	assert.Equal(t, sampleConfig.Volumes, actual.Volumes)
}

func TestParseAndLoad(t *testing.T) {
	actual, err := loadYAML(sampleYAML)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, serviceSort(sampleConfig.Services), serviceSort(actual.Services))
	assert.Equal(t, sampleConfig.Networks, actual.Networks)
	assert.Equal(t, sampleConfig.Volumes, actual.Volumes)
}

func TestInvalidTopLevelObjectType(t *testing.T) {
	_, err := loadYAML("1")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Top-level object must be a mapping")

	_, err = loadYAML("\"hello\"")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Top-level object must be a mapping")

	_, err = loadYAML("[\"hello\"]")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Top-level object must be a mapping")
}

func TestNonStringKeys(t *testing.T) {
	_, err := loadYAML(`
123:
  image: busybox
`)
	assert.NotNil(t, err)

	_, err = loadYAML(`
version: "2.1"
services:
  foo:
    image: busybox
  123:
    image: busybox
`)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Non-string key")

	_, err = loadYAML(`
version: "2.1"
services:
  foo:
    image: busybox
networks:
  default:
    ipam:
      config:
        - 123: oh dear
`)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Non-string key")
}

func loadYAML(yaml string) (*types.Config, error) {
	configFile, err := ParseYAML([]byte(yaml), "filename.yml")
	if err != nil {
		return nil, err
	}

	return Load(buildConfigDetails(configFile.Config))
}

func serviceSort(services []types.ServiceConfig) []types.ServiceConfig {
	sort.Sort(servicesByName(services))
	return services
}

type servicesByName []types.ServiceConfig

func (sbn servicesByName) Len() int           { return len(sbn) }
func (sbn servicesByName) Swap(i, j int)      { sbn[i], sbn[j] = sbn[j], sbn[i] }
func (sbn servicesByName) Less(i, j int) bool { return sbn[i].Name < sbn[j].Name }
