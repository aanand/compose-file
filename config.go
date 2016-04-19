package main

import (
	"fmt"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

type dict map[string]interface{}

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
}

type ServiceConfig struct {
	Name        string
	Image       string
	Environment map[string]string
}

func ParseYAML(source []byte, filename string) (*ConfigFile, error) {
	var cfg interface{}
	if err := yaml.Unmarshal(source, &cfg); err != nil {
		return nil, err
	}
	cfgDict, err := toDict(cfg)
	if err != nil {
		return nil, err
	}
	return &ConfigFile{Filename: filename, Config: cfgDict}, nil
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
		if servicesDict, ok := services.(dict); ok {
			for name, serviceCfg := range servicesDict {
				if serviceDict, ok := serviceCfg.(dict); ok {
					service, err := loadService(name, serviceDict)
					if err != nil {
						return nil, err
					}
					cfg.Services = append(cfg.Services, *service)
				} else {
					return nil, fmt.Errorf("services.%s must be a mapping, got: %#v", name, serviceCfg)
				}
			}
		} else {
			return nil, fmt.Errorf("services must be a mapping")
		}
	}

	return &cfg, nil
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

func parseMappingOrList(mappingOrList interface{}, sep, configKey string) (map[string]string, error) {
	result := make(map[string]string)

	if mapping, ok := mappingOrList.(dict); ok {
		for name, value := range mapping {
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

func toDict(val interface{}) (dict, error) {
	val, err := mapsToDicts(val)
	if err != nil {
		return nil, err
	}
	if d, ok := val.(dict); ok {
		return d, nil
	} else {
		return nil, fmt.Errorf("Expected dictionary at top level, got: %#v", val)
	}
}

func mapsToDicts(val interface{}) (interface{}, error) {
	if m, ok := val.(map[interface{}]interface{}); ok {
		d := make(dict)

		for key, value := range m {
			if str, ok := key.(string); ok {
				value, err := mapsToDicts(value)
				if err != nil {
					return nil, err
				}
				d[str] = value
			} else {
				return nil, fmt.Errorf("Expecting string key, got: %#v", key)
			}
		}

		return d, nil
	} else if s, ok := val.([]interface{}); ok {
		for idx, value := range s {
			value, err := mapsToDicts(value)
			if err != nil {
				return nil, err
			}
			s[idx] = value
		}

		return s, nil
	} else {
		return val, nil
	}
}
