package loader

import (
	"fmt"
	"strings"

	yaml "gopkg.in/yaml.v2"

	"github.com/aanand/compose-file/schema"
	"github.com/aanand/compose-file/types"
)

func ParseYAML(source []byte, filename string) (*types.ConfigFile, error) {
	var cfg interface{}
	if err := yaml.Unmarshal(source, &cfg); err != nil {
		return nil, err
	}
	cfgMap, ok := cfg.(map[interface{}]interface{})
	if !ok {
		return nil, fmt.Errorf("Top-level object must be a mapping")
	}
	converted, err := convertToStringKeysRecursive(cfgMap, "")
	if err != nil {
		return nil, err
	}
	configFile := types.ConfigFile{
		Filename: filename,
		Config:   converted.(types.Dict),
	}
	return &configFile, nil
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
		serviceMapping, err := loadServices(services.(types.Dict))
		if err != nil {
			return nil, err
		}
		cfg.Services = serviceMapping
	}

	if networks, ok := file.Config["networks"]; ok {
		networkMapping, err := loadNetworks(networks.(types.Dict))
		if err != nil {
			return nil, err
		}
		cfg.Networks = networkMapping
	}

	if volumes, ok := file.Config["volumes"]; ok {
		volumeMapping, err := loadVolumes(volumes.(types.Dict))
		if err != nil {
			return nil, err
		}
		cfg.Volumes = volumeMapping
	}

	return &cfg, nil
}

func validateAgainstConfigSchema(file types.ConfigFile) error {
	return schema.Validate(file.Config)
}

func convertToStringKeysRecursive(value interface{}, keyPrefix string) (interface{}, error) {
	if mapping, ok := value.(map[interface{}]interface{}); ok {
		dict := make(types.Dict)
		for key, entry := range mapping {
			str, ok := key.(string)
			if !ok {
				var location string
				if keyPrefix == "" {
					location = "at top level"
				} else {
					location = fmt.Sprintf("in %s", keyPrefix)
				}
				return nil, fmt.Errorf("Non-string key %s: %#v", location, key)
			}
			var newKeyPrefix string
			if keyPrefix == "" {
				newKeyPrefix = str
			} else {
				newKeyPrefix = fmt.Sprintf("%s.%s", keyPrefix, str)
			}
			convertedEntry, err := convertToStringKeysRecursive(entry, newKeyPrefix)
			if err != nil {
				return nil, err
			}
			dict[str] = convertedEntry
		}
		return dict, nil
	} else if list, ok := value.([]interface{}); ok {
		var convertedList []interface{}
		for index, entry := range list {
			newKeyPrefix := fmt.Sprintf("%s[%d]", keyPrefix, index)
			convertedEntry, err := convertToStringKeysRecursive(entry, newKeyPrefix)
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

func loadServices(servicesDict types.Dict) ([]types.ServiceConfig, error) {
	var services []types.ServiceConfig

	for name, serviceDef := range servicesDict {
		serviceConfig, err := loadService(name, serviceDef.(types.Dict))
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

	if image, ok := serviceDict["image"]; ok {
		service.Image = image.(string)
	}

	if environment, ok := serviceDict["environment"]; ok {
		service.Environment = loadMappingOrList(environment, "=")
	}

	return &service, nil
}

func loadNetworks(networksDict types.Dict) (map[string]types.NetworkConfig, error) {
	networks := make(map[string]types.NetworkConfig)

	for name, networkDef := range networksDict {
		networkConfig, err := loadNetwork(name, networkDef.(types.Dict))
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
		network.DriverOpts = loadStringMapping(driverOpts)
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

func loadVolumes(volumesDict types.Dict) (map[string]types.VolumeConfig, error) {
	volumes := make(map[string]types.VolumeConfig)

	for name, volumeDef := range volumesDict {
		volumeConfig, err := loadVolume(name, volumeDef.(types.Dict))
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
		volume.DriverOpts = loadStringMapping(driverOpts)
	}
	return &volume, nil
}

func loadStringMapping(value interface{}) map[string]string {
	mapping := value.(types.Dict)
	result := make(map[string]string)
	for name, item := range mapping {
		result[name] = item.(string)
	}
	return result
}

func loadMappingOrList(mappingOrList interface{}, sep string) map[string]string {
	result := make(map[string]string)

	if mapping, ok := mappingOrList.(types.Dict); ok {
		for name, value := range mapping {
			if value == nil {
				result[name] = ""
			} else {
				result[name] = fmt.Sprint(value)
			}
		}
	} else if list, ok := mappingOrList.([]interface{}); ok {
		for _, value := range list {
			parts := strings.SplitN(value.(string), sep, 2)
			if len(parts) == 1 {
				result[parts[0]] = ""
			} else {
				result[parts[0]] = parts[1]
			}
		}
	} else {
		panic(fmt.Errorf("expected a map or a slice, got: %#v", mappingOrList))
	}

	return result
}
