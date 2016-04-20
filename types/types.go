package types

type Dict map[interface{}]interface{}

type ConfigFile struct {
	Filename string
	Config   Dict
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
	Driver     string
	DriverOpts map[string]string
	IPAM       IPAMConfig
}

type IPAMConfig struct {
	Driver string
	Config []IPAMPool
}

type IPAMPool struct {
	Subnet string
}

type VolumeConfig struct {
	Driver     string
	DriverOpts map[string]string
}
