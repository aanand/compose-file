package loader

import (
	"io/ioutil"
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
			Ipam: types.IPAMConfig{
				Driver: "default",
				Config: []*types.IPAMPool{
					&types.IPAMPool{
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
	assert.NoError(t, err)
	assert.Equal(t, sampleDict, configFile.Config)
}

func TestLoad(t *testing.T) {
	actual, err := Load(buildConfigDetails(sampleDict))
	assert.NoError(t, err)
	assert.Equal(t, serviceSort(sampleConfig.Services), serviceSort(actual.Services))
	assert.Equal(t, sampleConfig.Networks, actual.Networks)
	assert.Equal(t, sampleConfig.Volumes, actual.Volumes)
}

func TestParseAndLoad(t *testing.T) {
	actual, err := loadYAML(sampleYAML)
	assert.NoError(t, err)
	assert.Equal(t, serviceSort(sampleConfig.Services), serviceSort(actual.Services))
	assert.Equal(t, sampleConfig.Networks, actual.Networks)
	assert.Equal(t, sampleConfig.Volumes, actual.Volumes)
}

func TestInvalidTopLevelObjectType(t *testing.T) {
	_, err := loadYAML("1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Top-level object must be a mapping")

	_, err = loadYAML("\"hello\"")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Top-level object must be a mapping")

	_, err = loadYAML("[\"hello\"]")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Top-level object must be a mapping")
}

func TestNonStringKeys(t *testing.T) {
	_, err := loadYAML(`
version: "2.1"
123:
  foo:
    image: busybox
`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Non-string key at top level: 123")

	_, err = loadYAML(`
version: "2.1"
services:
  foo:
    image: busybox
  123:
    image: busybox
`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Non-string key in services: 123")

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
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Non-string key in networks.default.ipam.config[0]: 123")

	_, err = loadYAML(`
version: "2.1"
services:
  dict-env:
    image: busybox
    environment:
      1: FOO
`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Non-string key in services.dict-env.environment: 1")
}

func TestUnsupportedVersion(t *testing.T) {
	_, err := loadYAML(`
version: "2"
services:
  foo:
    image: busybox
`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "version")

	_, err = loadYAML(`
version: "2.0"
services:
  foo:
    image: busybox
`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "version")
}

func TestInvalidVersion(t *testing.T) {
	_, err := loadYAML(`
version: 2.1
services:
  foo:
    image: busybox
`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "version must be a string")
}

func TestV1Unsupported(t *testing.T) {
	_, err := loadYAML(`
foo:
  image: busybox
`)
	assert.Error(t, err)
}

func TestNonMappingObject(t *testing.T) {
	_, err := loadYAML(`
version: "2.1"
services:
  - foo:
      image: busybox
`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "services must be a mapping")

	_, err = loadYAML(`
version: "2.1"
services:
  foo: busybox
`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "services.foo must be a mapping")

	_, err = loadYAML(`
version: "2.1"
networks:
  - default:
      driver: bridge
`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "networks must be a mapping")

	_, err = loadYAML(`
version: "2.1"
networks:
  default: bridge
`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "networks.default must be a mapping")

	_, err = loadYAML(`
version: "2.1"
volumes:
  - data:
      driver: local
`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "volumes must be a mapping")

	_, err = loadYAML(`
version: "2.1"
volumes:
  data: local
`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "volumes.data must be a mapping")
}

func TestNonStringImage(t *testing.T) {
	_, err := loadYAML(`
version: "2.1"
services:
  foo:
    image: ["busybox", "latest"]
`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "services.foo.image must be a string")
}

func TestValidEnvironment(t *testing.T) {
	config, err := loadYAML(`
version: "2.1"
services:
  dict-env:
    image: busybox
    environment:
      FOO: "1"
      BAR: 2
      BAZ: 2.5
      QUUX:
  list-env:
    image: busybox
    environment:
      - FOO=1
      - BAR=2
      - BAZ=2.5
      - QUUX=
`)
	assert.NoError(t, err)

	expected := map[string]string{
		"FOO":  "1",
		"BAR":  "2",
		"BAZ":  "2.5",
		"QUUX": "",
	}

	assert.Equal(t, 2, len(config.Services))

	for _, service := range config.Services {
		assert.Equal(t, expected, service.Environment)
	}
}

func TestInvalidEnvironmentValue(t *testing.T) {
	_, err := loadYAML(`
version: "2.1"
services:
  dict-env:
    image: busybox
    environment:
      FOO: ["1"]
`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "services.dict-env.environment.FOO must be a string, number or null")
}

func TestInvalidEnvironmentObject(t *testing.T) {
	_, err := loadYAML(`
version: "2.1"
services:
  dict-env:
    image: busybox
    environment: "FOO=1"
`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "services.dict-env.environment must be a mapping")
}

func TestFullExample(t *testing.T) {
	bytes, err := ioutil.ReadFile("full-example.yml")
	assert.NoError(t, err)

	config, err := loadYAML(string(bytes))
	assert.NoError(t, err)

	expectedServiceConfig := types.ServiceConfig{
		Name: "foo",

		CapAdd:        []string{"ALL"},
		CapDrop:       []string{"NET_ADMIN", "SYS_ADMIN"},
		CgroupParent:  "m-executor-abcd",
		Command:       []string{"bundle", "exec", "thin", "-p", "3000"},
		ContainerName: "my-web-container",
		DependsOn:     []string{"db", "redis"},
		Devices:       []string{"/dev/ttyUSB0:/dev/ttyUSB0"},
		Dns:           []string{"8.8.8.8", "9.9.9.9"},
		DnsSearch:     []string{"dc1.example.com", "dc2.example.com"},
		Entrypoint:    []string{"/code/entrypoint.sh", "-p", "3000"},
		Environment: map[string]string{
			"RACK_ENV":       "development",
			"SHOW":           "true",
			"SESSION_SECRET": "",
		},
		Expose: []string{"3000", "8000"},
		ExternalLinks: []string{
			"redis_1",
			"project_db_1:mysql",
			"project_db_1:postgresql",
		},
		ExtraHosts: map[string]string{
			"otherhost": "50.31.209.229",
			"somehost":  "162.242.195.82",
		},
		Hostname: "foo",
		Image:    "redis",
		Ipc:      "host",
		Labels: map[string]string{
			"com.example.description": "Accounting webapp",
			"com.example.number":      "42",
			"com.example.empty-label": "",
		},
		Links: []string{
			"db",
			"db:database",
			"redis",
		},
		Logging: types.LoggingConfig{
			Driver: "syslog",
			Options: map[string]string{
				"syslog-address": "tcp://192.168.0.42:123",
			},
		},
		MacAddress:   "02:42:ac:11:65:43",
		MemLimit:     1073741824,
		MemswapLimit: 2147483648,
		NetworkMode:  "container:0cfeab0f748b9a743dc3da582046357c6ef497631c1a016d28d2bf9b4f899f7b",
		Networks: map[string]*types.ServiceNetworkConfig{
			"some-network": &types.ServiceNetworkConfig{
				Aliases:     []string{"alias1", "alias3"},
				Ipv4Address: "",
				Ipv6Address: "",
			},
			"other-network": &types.ServiceNetworkConfig{
				Aliases:     nil,
				Ipv4Address: "172.16.238.10",
				Ipv6Address: "2001:3984:3989::10",
			},
			"other-other-network": &types.ServiceNetworkConfig{},
		},
		Pid: "host",
		Ports: []string{
			"3000",
			"3000-3005",
			"8000:8000",
			"9090-9091:8080-8081",
			"49100:22",
			"127.0.0.1:8001:8001",
			"127.0.0.1:5000-5010:5000-5010",
		},
		Privileged: true,
		ReadOnly:   true,
		Restart:    "always",
		SecurityOpt: []string{
			"label=level:s0:c100,c200",
			"label=type:svirt_apache_t",
		},
		ShmSize:    67108864,
		StdinOpen:  true,
		StopSignal: "SIGUSR1",
		Tmpfs:      []string{"/run", "/tmp"},
		Tty:        true,
		Ulimits: map[string]*types.UlimitsConfig{
			"nproc": &types.UlimitsConfig{
				Single: 65535,
			},
			"nofile": &types.UlimitsConfig{
				Soft: 20000,
				Hard: 40000,
			},
		},
		User: "someone",
		Volumes: []string{
			"/var/lib/mysql",
			"/opt/data:/var/lib/mysql",
			".:/code",
			"./static:/var/www/html",
			"~/configs:/etc/configs/:ro",
			"datavolume:/var/lib/mysql",
		},
		VolumeDriver: "mydriver",
		WorkingDir:   "/code",
	}

	assert.Equal(t, []types.ServiceConfig{expectedServiceConfig}, config.Services)

	expectedNetworkConfig := map[string]types.NetworkConfig{
		"some-network": types.NetworkConfig{},

		"other-network": types.NetworkConfig{
			Driver: "overlay",
			DriverOpts: map[string]string{
				"foo": "bar",
				"baz": "1",
			},
			Ipam: types.IPAMConfig{
				Driver: "overlay",
				Config: []*types.IPAMPool{
					&types.IPAMPool{Subnet: "172.16.238.0/24"},
					&types.IPAMPool{Subnet: "2001:3984:3989::/64"},
				},
			},
		},

		"external-network": types.NetworkConfig{
			ExternalName: "external-network",
		},

		"other-external-network": types.NetworkConfig{
			ExternalName: "my-cool-network",
		},
	}

	assert.Equal(t, expectedNetworkConfig, config.Networks)

	expectedVolumeConfig := map[string]types.VolumeConfig{
		"some-volume": types.VolumeConfig{},
		"other-volume": types.VolumeConfig{
			Driver: "flocker",
			DriverOpts: map[string]string{
				"foo": "bar",
				"baz": "1",
			},
		},
		"external-volume": types.VolumeConfig{
			ExternalName: "external-volume",
		},
		"other-external-volume": types.VolumeConfig{
			ExternalName: "my-cool-volume",
		},
	}

	assert.Equal(t, expectedVolumeConfig, config.Volumes)
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
