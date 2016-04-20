package loader

import (
	"fmt"
	"strings"

	yaml "gopkg.in/yaml.v2"

	"github.com/aanand/compose-file/schema"
	"github.com/aanand/compose-file/types"
)

func ParseYAML(source []byte, filename string) (*types.ConfigFile, error) {
	var cfg types.Dict
	if err := yaml.Unmarshal(source, &cfg); err != nil {
		return nil, err
	}
	return &types.ConfigFile{Filename: filename, Config: cfg}, nil
}

func Load(configDetails types.ConfigDetails) (*types.Config, error) {
	if len(configDetails.ConfigFiles) < 1 {
		return nil, fmt.Errorf("No files specified")
	}
	if len(configDetails.ConfigFiles) > 1 {
		return nil, fmt.Errorf("Multiple files are not yet supported")
	}

	cfg := types.Config{}
	file := configDetails.ConfigFiles[0]

	if err := validateAgainstConfigSchema(file); err != nil {
		return nil, err
	}

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

func validateAgainstConfigSchema(file types.ConfigFile) error {
	config, err := validateStringKeys(file.Config)
	if err != nil {
		return err
	}
	return schema.Validate(config)
}

func validateStringKeys(config types.Dict) (map[string]interface{}, error) {
	converted, err := convertToStringKeysRecursive(config)
	if err != nil {
		return nil, err
	}
	configMap, ok := converted.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Top-level object must be a mapping")
	}
	return configMap, nil
}

func convertToStringKeysRecursive(value interface{}) (interface{}, error) {
	if dict, ok := value.(types.Dict); ok {
		convertedDict := make(map[string]interface{})
		for key, entry := range dict {
			str, ok := key.(string)
			if !ok {
				return nil, fmt.Errorf("Non-string key: %#v", key)
			}
			convertedEntry, err := convertToStringKeysRecursive(entry)
			if err != nil {
				return nil, err
			}
			convertedDict[str] = convertedEntry
		}
		return convertedDict, nil
	} else if list, ok := value.([]interface{}); ok {
		var convertedList []interface{}
		for _, entry := range list {
			convertedEntry, err := convertToStringKeysRecursive(entry)
			if err != nil {
				return nil, err
			}
			convertedList = append(convertedList, convertedEntry)
		}
		return convertedList, nil
	} else {
		return value, nil
	}
}

func loadServices(value interface{}) ([]types.ServiceConfig, error) {
	servicesDict, ok := value.(types.Dict)
	if !ok {
		return nil, fmt.Errorf("services must be a mapping")
	}

	var services []types.ServiceConfig

	for key, serviceDef := range servicesDict {
		name, ok := key.(string)
		if !ok {
			return nil, fmt.Errorf("services contains a non-string key (%#v)", key)
		}
		serviceDict, ok := serviceDef.(types.Dict)
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

func loadService(name string, serviceDict types.Dict) (*types.ServiceConfig, error) {
	service := types.ServiceConfig{}
	service.Name = name

	if image, ok := serviceDict["image"].(string); ok {
		service.Image = image
	} else {
		return nil, fmt.Errorf("services.%s.image must be a string, got: %#v", serviceDict["image"])
	}

	if environment, ok := serviceDict["environment"]; ok {
		envMap, err := loadMappingOrList(environment, "=", fmt.Sprintf("services.%s.environment", name))
		if err != nil {
			return nil, err
		}
		service.Environment = envMap
	}

	return &service, nil
}

func loadNetworks(value interface{}) (map[string]types.NetworkConfig, error) {
	networksDict, ok := value.(types.Dict)
	if !ok {
		return nil, fmt.Errorf("networks must be a mapping")
	}

	networks := make(map[string]types.NetworkConfig)

	for key, networkDef := range networksDict {
		name, ok := key.(string)
		if !ok {
			return nil, fmt.Errorf("networks contains a non-string key (%#v)", key)
		}
		networkDict, ok := networkDef.(types.Dict)
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

func loadNetwork(name string, networkDict types.Dict) (*types.NetworkConfig, error) {
	network := types.NetworkConfig{}
	if driver, ok := networkDict["driver"]; ok {
		network.Driver = driver.(string)
	}
	if driverOpts, ok := networkDict["driver_opts"]; ok {
		network.DriverOpts = loadStringMappingUnsafe(driverOpts)
	}
	if ipam, ok := networkDict["ipam"]; ok {
		network.IPAM = loadIPAMConfig(ipam.(types.Dict))
	}
	return &network, nil
}

func loadIPAMConfig(ipamDict types.Dict) types.IPAMConfig {
	ipamConfig := types.IPAMConfig{}
	if driver, ok := ipamDict["driver"]; ok {
		ipamConfig.Driver = driver.(string)
	}
	if config, ok := ipamDict["config"]; ok {
		for _, poolDef := range config.([]interface{}) {
			ipamConfig.Config = append(ipamConfig.Config, loadIPAMPool(poolDef.(types.Dict)))
		}
	}
	return ipamConfig
}

func loadIPAMPool(poolDict types.Dict) types.IPAMPool {
	ipamPool := types.IPAMPool{}
	if subnet, ok := poolDict["subnet"]; ok {
		ipamPool.Subnet = subnet.(string)
	}
	return ipamPool
}

func loadVolumes(value interface{}) (map[string]types.VolumeConfig, error) {
	volumesDict, ok := value.(types.Dict)
	if !ok {
		return nil, fmt.Errorf("volumes must be a mapping")
	}

	volumes := make(map[string]types.VolumeConfig)

	for key, volumeDef := range volumesDict {
		name, ok := key.(string)
		if !ok {
			return nil, fmt.Errorf("volumes contains a non-string key (%#v)", key)
		}
		volumeDict, ok := volumeDef.(types.Dict)
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

func loadVolume(name string, volumeDict types.Dict) (*types.VolumeConfig, error) {
	volume := types.VolumeConfig{}
	if driver, ok := volumeDict["driver"].(string); ok {
		volume.Driver = driver
	}
	if driverOpts, ok := volumeDict["driver_opts"]; ok {
		volume.DriverOpts = loadStringMappingUnsafe(driverOpts)
	}
	return &volume, nil
}

func loadStringMappingUnsafe(value interface{}) map[string]string {
	mapping := value.(types.Dict)
	result := make(map[string]string)
	for key, item := range mapping {
		result[key.(string)] = item.(string)
	}
	return result
}

func loadMappingOrList(mappingOrList interface{}, sep, configKey string) (map[string]string, error) {
	result := make(map[string]string)

	if mapping, ok := mappingOrList.(types.Dict); ok {
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
