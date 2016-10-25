package types

type Dict map[string]interface{}

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
	Name string

	CapAdd          []string
	CapDrop         []string
	CgroupParent    string
	Command         []string `compose:"shell_command"`
	ContainerName   string
	DependsOn       []string
	Deploy          DeployConfig
	Devices         []string
	Dns             []string `compose:"string_or_list"`
	DnsSearch       []string `compose:"string_or_list"`
	DomainName      string
	Entrypoint      []string          `compose:"shell_command"`
	Environment     map[string]string `compose:"list_or_dict_equals"`
	Expose          []string          `compose:"list_of_strings_or_numbers"`
	ExternalLinks   []string
	ExtraHosts      map[string]string `compose:"list_or_dict_colon"`
	Hostname        string
	Image           string
	Ipc             string
	Labels          map[string]string `compose:"list_or_dict_equals"`
	Links           []string
	Logging         *LoggingConfig
	MacAddress      string
	MemLimit        int `compose:"size"`
	MemswapLimit    int `compose:"size"`
	NetworkMode     string
	Networks        map[string]*ServiceNetworkConfig `compose:"list_or_struct_map"`
	Pid             string
	Ports           []string `compose:"list_of_strings_or_numbers"`
	Privileged      bool
	ReadOnly        bool
	Restart         string
	SecurityOpt     []string
	ShmSize         int `compose:"size"`
	StdinOpen       bool
	StopGracePeriod *string
	StopSignal      string
	Tmpfs           []string `compose:"string_or_list"`
	Tty             bool
	Ulimits         map[string]*UlimitsConfig `compose:"-"`
	User            string
	Volumes         []string
	VolumeDriver    string
	WorkingDir      string
}

type LoggingConfig struct {
	Driver  string
	Options map[string]string
}

type DeployConfig struct {
	Mode      string
	Replicas  uint64
	Labels    map[string]string `compose:"list_or_dict_equals"`
	Placement Placement
}

type Placement struct {
	Constraints []string
}

type ServiceNetworkConfig struct {
	Aliases     []string
	Ipv4Address string
	Ipv6Address string
}

type UlimitsConfig struct {
	Single int
	Soft   int
	Hard   int
}

type NetworkConfig struct {
	Driver       string
	DriverOpts   map[string]string
	Ipam         IPAMConfig
	ExternalName string
	Labels       map[string]string `compose:"list_or_dict_equals"`
}

type IPAMConfig struct {
	Driver string
	Config []*IPAMPool
}

type IPAMPool struct {
	Subnet string
}

type VolumeConfig struct {
	Driver       string
	DriverOpts   map[string]string
	ExternalName string
	Labels       map[string]string `compose:"list_or_dict_equals"`
}
