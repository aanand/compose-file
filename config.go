package main

import (
	"fmt"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

type dict map[interface{}]interface{}

type ConfigFile struct {
	Filename string
	Config   dict
}

type ConfigDetails struct {
	WorkingDir  string
	ConfigFiles []ConfigFile
	Environment map[string]string
}

type Config struct {
	Services []ServiceConfig
	Networks map[string]NetworkConfig
	Volumes  map[string]VolumeConfig
}

type ServiceConfig struct {
	Name        string
	Image       string
	Environment map[string]string
}

type NetworkConfig struct {
	Driver string
}

type VolumeConfig struct {
	Driver string
}

func ParseYAML(source []byte, filename string) (*ConfigFile, error) {
	var cfg dict
	if err := yaml.Unmarshal(source, &cfg); err != nil {
		return nil, err
	}
	return &ConfigFile{Filename: filename, Config: cfg}, nil
}

func Load(configDetails ConfigDetails) (*Config, error) {
	if len(configDetails.ConfigFiles) < 1 {
		return nil, fmt.Errorf("No files specified")
	}
	if len(configDetails.ConfigFiles) > 1 {
		return nil, fmt.Errorf("Multiple files are not yet supported")
	}

	cfg := Config{}
	file := configDetails.ConfigFiles[0]

	if services, ok := file.Config["services"]; ok {
		serviceMapping, err := loadServices(services)
		if err != nil {
			return nil, err
		}
		cfg.Services = serviceMapping
	}

	if networks, ok := file.Config["networks"]; ok {
		networkMapping, err := loadNetworks(networks)
		if err != nil {
			return nil, err
		}
		cfg.Networks = networkMapping
	}

	if volumes, ok := file.Config["volumes"]; ok {
		volumeMapping, err := loadVolumes(volumes)
		if err != nil {
			return nil, err
		}
		cfg.Volumes = volumeMapping
	}

	return &cfg, nil
}

func loadServices(value interface{}) ([]ServiceConfig, error) {
	servicesDict, ok := value.(dict)
	if !ok {
		return nil, fmt.Errorf("services must be a mapping")
	}

	var services []ServiceConfig

	for key, serviceDef := range servicesDict {
		name, ok := key.(string)
		if !ok {
			return nil, fmt.Errorf("services contains a non-string key (%#v)", key)
		}
		serviceDict, ok := serviceDef.(dict)
		if !ok {
			return nil, fmt.Errorf("services.%s must be a mapping, got: %#v", name, serviceDef)
		}
		serviceConfig, err := loadService(name, serviceDict)
		if err != nil {
			return nil, err
		}
		services = append(services, *serviceConfig)
	}

	return services, nil
}

func loadService(name string, serviceDict dict) (*ServiceConfig, error) {
	service := ServiceConfig{}
	service.Name = name

	if image, ok := serviceDict["image"].(string); ok {
		service.Image = image
	} else {
		return nil, fmt.Errorf("services.%s.image must be a string, got: %#v", serviceDict["image"])
	}

	if environment, ok := serviceDict["environment"]; ok {
		envMap, err := parseMappingOrList(environment, "=", fmt.Sprintf("services.%s.environment", name))
		if err != nil {
			return nil, err
		}
		service.Environment = envMap
	}

	return &service, nil
}

func loadNetworks(value interface{}) (map[string]NetworkConfig, error) {
	networksDict, ok := value.(dict)
	if !ok {
		return nil, fmt.Errorf("networks must be a mapping")
	}

	networks := make(map[string]NetworkConfig)

	for key, networkDef := range networksDict {
		name, ok := key.(string)
		if !ok {
			return nil, fmt.Errorf("networks contains a non-string key (%#v)", key)
		}
		networkDict, ok := networkDef.(dict)
		if !ok {
			return nil, fmt.Errorf("networks.%s must be a mapping, got: %#v", name, networkDef)
		}
		networkConfig, err := loadNetwork(name, networkDict)
		if err != nil {
			return nil, err
		}
		networks[name] = *networkConfig
	}

	return networks, nil
}

func loadNetwork(name string, networkDict dict) (*NetworkConfig, error) {
	network := NetworkConfig{}
	if driver, ok := networkDict["driver"].(string); ok {
		network.Driver = driver
	}
	return &network, nil
}

func loadVolumes(value interface{}) (map[string]VolumeConfig, error) {
	volumesDict, ok := value.(dict)
	if !ok {
		return nil, fmt.Errorf("volumes must be a mapping")
	}

	volumes := make(map[string]VolumeConfig)

	for key, volumeDef := range volumesDict {
		name, ok := key.(string)
		if !ok {
			return nil, fmt.Errorf("volumes contains a non-string key (%#v)", key)
		}
		volumeDict, ok := volumeDef.(dict)
		if !ok {
			return nil, fmt.Errorf("volumes.%s must be a mapping, got: %#v", name, volumeDef)
		}
		volumeConfig, err := loadVolume(name, volumeDict)
		if err != nil {
			return nil, err
		}
		volumes[name] = *volumeConfig
	}

	return volumes, nil
}

func loadVolume(name string, volumeDict dict) (*VolumeConfig, error) {
	volume := VolumeConfig{}
	if driver, ok := volumeDict["driver"].(string); ok {
		volume.Driver = driver
	}
	return &volume, nil
}

func parseMappingOrList(mappingOrList interface{}, sep, configKey string) (map[string]string, error) {
	result := make(map[string]string)

	if mapping, ok := mappingOrList.(dict); ok {
		for key, value := range mapping {
			name, ok := key.(string)
			if !ok {
				return nil, fmt.Errorf("%s contains a non-string key (%#v)", configKey, key)
			}
			if str, ok := value.(string); ok {
				result[name] = str
			} else {
				return nil, fmt.Errorf("%s.%s has non-string value: %#v", configKey, name, value)
			}
		}
	} else if list, ok := mappingOrList.([]interface{}); ok {
		for _, value := range list {
			if str, ok := value.(string); ok {
				parts := strings.SplitN(str, sep, 2)
				if len(parts) == 1 {
					result[parts[0]] = ""
				} else {
					result[parts[0]] = parts[1]
				}
			} else {
				return nil, fmt.Errorf("%s has a non-string item: %#v", configKey, value)
			}
		}
	} else {
		return nil, fmt.Errorf("%s must be a mapping or a list, got: %#v", configKey, mappingOrList)
	}

	return result, nil
}
