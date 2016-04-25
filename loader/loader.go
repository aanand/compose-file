package loader

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	shellwords "github.com/mattn/go-shellwords"
	yaml "gopkg.in/yaml.v2"

	"github.com/aanand/compose-file/schema"
	"github.com/aanand/compose-file/types"
)

var fieldNameRegexp *regexp.Regexp

func init() {
	r, err := regexp.Compile("[[:upper:]][[:lower:]]+")
	if err != nil {
		panic(err)
	}
	fieldNameRegexp = r
}

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

	version := file.Config["version"].(string)
	if version != "2.1" {
		return nil, fmt.Errorf("Unsupported version: %#v. The only supported version is 2.1", version)
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
	serviceType := reflect.TypeOf(types.ServiceConfig{})
	serviceValue := reflect.New(serviceType).Elem()
	serviceValue.FieldByName("Name").SetString(name)

	for i := 0; i < serviceType.NumField(); i++ {
		field := serviceType.Field(i)
		fieldValue := serviceValue.FieldByIndex([]int{i})
		fieldTag := field.Tag.Get("compose")

		yamlName := toYAMLName(field.Name)
		value, ok := serviceDict[yamlName]
		if !ok {
			continue
		}

		fmt.Println(yamlName)

		if fieldTag == "list_or_dict_equals" {
			fieldValue.Set(reflect.ValueOf(loadMappingOrList(value, "=")))
		} else if fieldTag == "string_or_list" {
			fieldValue.Set(reflect.ValueOf(loadStringOrListOfStrings(value)))
		} else if fieldTag == "list_of_strings_or_numbers" {
			fieldValue.Set(reflect.ValueOf(loadListOfStringsOrNumbers(value)))
		} else if fieldTag == "shell_command" {
			command, err := loadShellCommand(value)
			if err != nil {
				return nil, err
			}
			fieldValue.Set(reflect.ValueOf(command))
		} else if fieldTag != "" {
			fmt.Printf("skipping %s - unrecognised tag %s\n", yamlName, fieldTag)
		} else if field.Type.Kind() == reflect.String {
			fieldValue.SetString(value.(string))
		} else if field.Type.Kind() == reflect.Slice && field.Type.Elem().Kind() == reflect.String {
			fieldValue.Set(reflect.ValueOf(loadListOfStrings(value)))
		}
	}

	serviceConfig := serviceValue.Interface().(types.ServiceConfig)
	return &serviceConfig, nil
}

func toYAMLName(name string) string {
	nameParts := fieldNameRegexp.FindAllString(name, -1)
	for i, p := range nameParts {
		nameParts[i] = strings.ToLower(p)
	}
	return strings.Join(nameParts, "_")
}

func loadNetworks(networksDict types.Dict) (map[string]types.NetworkConfig, error) {
	networks := make(map[string]types.NetworkConfig)

	for name, networkDef := range networksDict {
		if networkDef == nil {
			networks[name] = types.NetworkConfig{}
		} else {
			networkConfig, err := loadNetwork(name, networkDef.(types.Dict))
			if err != nil {
				return nil, err
			}
			networks[name] = *networkConfig
		}
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
		if volumeDef == nil {
			volumes[name] = types.VolumeConfig{}
		} else {
			volumeConfig, err := loadVolume(name, volumeDef.(types.Dict))
			if err != nil {
				return nil, err
			}
			volumes[name] = *volumeConfig
		}
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
		result[name] = toString(item)
	}
	return result
}

func loadListOfStrings(value interface{}) []string {
	list := value.([]interface{})
	result := make([]string, len(list))
	for i, item := range list {
		result[i] = item.(string)
	}
	return result
}

func loadListOfStringsOrNumbers(value interface{}) []string {
	list := value.([]interface{})
	result := make([]string, len(list))
	for i, item := range list {
		result[i] = fmt.Sprint(item)
	}
	return result
}

func loadStringOrListOfStrings(value interface{}) []string {
	if _, ok := value.([]interface{}); ok {
		return loadListOfStrings(value)
	} else {
		return []string{value.(string)}
	}
}

func loadMappingOrList(mappingOrList interface{}, sep string) map[string]string {
	result := make(map[string]string)

	if mapping, ok := mappingOrList.(types.Dict); ok {
		for name, value := range mapping {
			result[name] = toString(value)
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

func loadShellCommand(value interface{}) ([]string, error) {
	if str, ok := value.(string); ok {
		return shellwords.Parse(str)
	} else {
		return loadListOfStrings(value), nil
	}
}

func toString(value interface{}) string {
	if value == nil {
		return ""
	} else {
		return fmt.Sprint(value)
	}
}
