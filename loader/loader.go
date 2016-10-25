package loader

import (
	"fmt"
	"os"
	"path"
	"reflect"
	"regexp"
	"strings"

	units "github.com/docker/go-units"
	shellwords "github.com/mattn/go-shellwords"
	yaml "gopkg.in/yaml.v2"

	"github.com/aanand/compose-file/schema"
	"github.com/aanand/compose-file/types"
)

var (
	fieldNameRegexp = regexp.MustCompile("[A-Z][a-z0-9]+")
)

// ParseYAML reads the bytes from a file, parses the bytes into a mapping
// structure, and returns it.
func ParseYAML(source []byte) (types.Dict, error) {
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
	return converted.(types.Dict), nil
}

// Load reads a ConfigDetails and returns a fully loaded configuration
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
	if version != "3" {
		return nil, fmt.Errorf("Unsupported version: %#v. The only supported version is 3", version)
	}

	if services, ok := file.Config["services"]; ok {
		serviceMapping, err := loadServices(services.(types.Dict), configDetails.WorkingDir)
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

// TODO: should this be renamed to validateStringKeys? Why do the keys need to
// be converted?
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
	}
	if list, ok := value.([]interface{}); ok {
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
	}
	return value, nil
}

func loadServices(servicesDict types.Dict, workingDir string) ([]types.ServiceConfig, error) {
	var services []types.ServiceConfig

	for name, serviceDef := range servicesDict {
		serviceConfig, err := loadService(name, serviceDef.(types.Dict), workingDir)
		if err != nil {
			return nil, err
		}
		services = append(services, *serviceConfig)
	}

	return services, nil
}

func loadService(name string, serviceDict types.Dict, workingDir string) (*types.ServiceConfig, error) {
	serviceConfig := &types.ServiceConfig{}
	if err := loadStruct(serviceDict, serviceConfig); err != nil {
		return nil, err
	}
	serviceConfig.Name = name

	// Load ulimits manually
	if ulimits, ok := serviceDict["ulimits"]; ok {
		serviceConfig.Ulimits = loadUlimits(ulimits)
	}

	if err := resolveVolumePaths(serviceConfig.Volumes, workingDir); err != nil {
		return nil, err
	}

	return serviceConfig, nil
}

// TODO: handle invalid mappings here?
func resolveVolumePaths(volumes []string, workingDir string) error {
	for i, mapping := range volumes {
		parts := strings.SplitN(mapping, ":", 2)
		if len(parts) == 1 {
			continue
		}

		if strings.HasPrefix(parts[0], ".") {
			parts[0] = path.Join(workingDir, parts[0])
		}
		parts[0] = expandUser(parts[0])

		volumes[i] = strings.Join(parts, ":")
	}

	return nil
}

// TODO: make this more robust
func expandUser(path string) string {
	if strings.HasPrefix(path, "~") {
		return strings.Replace(path, "~", os.Getenv("HOME"), 1)
	}
	return path
}

// TODO: this should be part of the transform
func loadUlimits(value interface{}) map[string]*types.UlimitsConfig {
	ulimitsMap := make(map[string]*types.UlimitsConfig)

	for name, item := range value.(types.Dict) {
		config := &types.UlimitsConfig{}
		if singleLimit, ok := item.(int); ok {
			config.Single = singleLimit
		} else {
			limitDict := item.(types.Dict)
			config.Soft = limitDict["soft"].(int)
			config.Hard = limitDict["hard"].(int)
		}
		ulimitsMap[name] = config
	}

	return ulimitsMap
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
	network := &types.NetworkConfig{}
	if err := loadStruct(networkDict, network); err != nil {
		return nil, err
	}
	if external, ok := networkDict["external"]; ok {
		network.ExternalName = loadExternalName(name, external)
	}
	return network, nil
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
	volume := &types.VolumeConfig{}
	if err := loadStruct(volumeDict, volume); err != nil {
		return nil, err
	}
	if external, ok := volumeDict["external"]; ok {
		volume.ExternalName = loadExternalName(name, external)
	}
	return volume, nil
}

func loadExternalName(resourceName string, value interface{}) string {
	if externalBool, ok := value.(bool); ok {
		if externalBool {
			return resourceName
		}
		return ""
	}
	return value.(types.Dict)["name"].(string)
}

func loadStruct(dict types.Dict, dest interface{}) error {
	structValue := reflect.ValueOf(dest).Elem()
	structType := structValue.Type()

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		fieldValue := structValue.FieldByIndex([]int{i})
		fieldTag := field.Tag.Get("compose")

		yamlName := toYAMLName(field.Name)
		value, ok := dict[yamlName]
		if !ok {
			continue
		}

		if fieldTag == "list_or_dict_equals" {
			fieldValue.Set(reflect.ValueOf(loadMappingOrList(value, "=")))
		} else if fieldTag == "list_or_dict_colon" {
			fieldValue.Set(reflect.ValueOf(loadMappingOrList(value, ":")))
		} else if fieldTag == "list_or_struct_map" {
			if err := loadListOrStructMap(value, fieldValue); err != nil {
				return err
			}
		} else if fieldTag == "string_or_list" {
			fieldValue.Set(reflect.ValueOf(loadStringOrListOfStrings(value)))
		} else if fieldTag == "list_of_strings_or_numbers" {
			fieldValue.Set(reflect.ValueOf(loadListOfStringsOrNumbers(value)))
		} else if fieldTag == "shell_command" {
			command, err := loadShellCommand(value)
			if err != nil {
				return err
			}
			fieldValue.Set(reflect.ValueOf(command))
		} else if fieldTag == "size" {
			size, err := loadSize(value)
			if err != nil {
				return err
			}
			fieldValue.SetInt(size)
		} else if fieldTag == "-" {
			// skip
		} else if fieldTag != "" {
			panic(fmt.Sprintf("Unrecognised field tag on %s: %s\n", field.Name, fieldTag))
		} else if field.Type.Kind() == reflect.String {
			fieldValue.SetString(value.(string))
		} else if field.Type.Kind() == reflect.Bool {
			fieldValue.SetBool(value.(bool))
		} else if field.Type.Kind() == reflect.Slice && field.Type.Elem().Kind() == reflect.String {
			fieldValue.Set(reflect.ValueOf(loadListOfStrings(value)))
		} else if field.Type.Kind() == reflect.Slice && field.Type.Elem().Kind() == reflect.Ptr && field.Type.Elem().Elem().Kind() == reflect.Struct {
			if err := loadListOfStructs(value, fieldValue); err != nil {
				return err
			}
		} else if field.Type.Kind() == reflect.Map && field.Type.Elem().Kind() == reflect.String {
			fieldValue.Set(reflect.ValueOf(loadStringMapping(value)))
		} else if field.Type.Kind() == reflect.Struct {
			if err := loadStruct(value.(types.Dict), fieldValue.Addr().Interface()); err != nil {
				return err
			}
		} else {
			panic(fmt.Sprintf("Can't load %s (%s): don't know how to load %v",
				field.Name, yamlName, field.Type))
		}
	}

	return nil
}

func toYAMLName(name string) string {
	nameParts := fieldNameRegexp.FindAllString(name, -1)
	for i, p := range nameParts {
		nameParts[i] = strings.ToLower(p)
	}
	return strings.Join(nameParts, "_")
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

func loadListOfStructs(value interface{}, dest reflect.Value) error {
	result := dest
	listOfDicts := value.([]interface{})
	for _, item := range listOfDicts {
		itemStruct := reflect.New(dest.Type().Elem().Elem())
		if err := loadStruct(item.(types.Dict), itemStruct.Interface()); err != nil {
			return err
		}
		result = reflect.Append(result, itemStruct)
	}
	dest.Set(result)
	return nil
}

func loadListOrStructMap(value interface{}, dest reflect.Value) error {
	mapValue := reflect.MakeMap(dest.Type())

	if list, ok := value.([]interface{}); ok {
		for _, name := range list {
			mapValue.SetMapIndex(reflect.ValueOf(name), reflect.ValueOf(nil))
		}
	} else {
		for name, item := range value.(types.Dict) {
			itemStruct := reflect.New(dest.Type().Elem().Elem())
			if item != nil {
				if err := loadStruct(item.(types.Dict), itemStruct.Interface()); err != nil {
					return err
				}
			}
			mapValue.SetMapIndex(reflect.ValueOf(name), itemStruct)
		}
	}

	dest.Set(mapValue)

	return nil
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
	}
	return []string{value.(string)}
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
	}
	return loadListOfStrings(value), nil
}

func loadSize(value interface{}) (int64, error) {
	if size, ok := value.(int); ok {
		return int64(size), nil
	}
	return units.RAMInBytes(value.(string))
}

func toString(value interface{}) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}
