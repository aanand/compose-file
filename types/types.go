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

	CapAdd          []string `mapstructure:"cap_add"`
	CapDrop         []string `mapstructure:"cap_drop"`
	CgroupParent    string   `mapstructure:"cgroup_parent"`
	Command         []string `compose:"shell_command"`
	ContainerName   string   `mapstructure:"container_name"`
	DependsOn       []string `mapstructure:"depends_on"`
	Deploy          DeployConfig
	Devices         []string
	Dns             []string          `compose:"string_or_list"`
	DnsSearch       []string          `mapstructure:"dns_search" compose:"string_or_list"`
	DomainName      string            `mapstructure:"domainname"`
	Entrypoint      []string          `compose:"shell_command"`
	Environment     map[string]string `compose:"list_or_dict_equals"`
	EnvFile         []string          `mapstructure:"env_file"`
	Expose          []string          `compose:"list_of_strings_or_numbers"`
	ExternalLinks   []string          `mapstructure:"external_links"`
	ExtraHosts      map[string]string `mapstructure:"extra_hosts" compose:"list_or_dict_colon"`
	Hostname        string
	Image           string
	Ipc             string
	Labels          map[string]string `compose:"list_or_dict_equals"`
	Links           []string
	Logging         *LoggingConfig
	MacAddress      string                           `mapstructure:"mac_address"`
	MemLimit        int64                            `mapstructure:"mem_limit" compose:"size"`
	MemswapLimit    int64                            `mapstructure:"memswap_limit" compose:"size"`
	NetworkMode     string                           `mapstructure:"network_mode"`
	Networks        map[string]*ServiceNetworkConfig `compose:"list_or_struct_map"`
	Pid             string
	Ports           []string `compose:"list_of_strings_or_numbers"`
	Privileged      bool
	ReadOnly        bool `mapstructure:"read_only"`
	Restart         string
	SecurityOpt     []string `mapstructure:"security_opt"`
	ShmSize         int64    `mapstructure:"shm_size" compose:"size"`
	StdinOpen       bool     `mapstructure:"stdin_open"`
	StopGracePeriod *string  `mapstructure:"stop_grace_period"`
	StopSignal      string   `mapstructure:"stop_signal"`
	Tmpfs           []string `compose:"string_or_list"`
	Tty             bool
	Ulimits         map[string]*UlimitsConfig
	User            string
	Volumes         []string
	VolumeDriver    string `mapstructure:"volume_driver"`
	WorkingDir      string `mapstructure:"working_dir"`
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
	Ipv4Address string `mapstructure:"ipv4_address"`
	Ipv6Address string `mapstructure:"ipv6_address"`
}

type UlimitsConfig struct {
	Single int
	Soft   int
	Hard   int
}

type NetworkConfig struct {
	Driver     string
	DriverOpts map[string]string `mapstructure:"driver_opts"`
	Ipam       IPAMConfig
	External   External
	Labels     map[string]string `compose:"list_or_dict_equals"`
}

type IPAMConfig struct {
	Driver string
	Config []*IPAMPool
}

type IPAMPool struct {
	Subnet string
}

type VolumeConfig struct {
	Driver     string
	DriverOpts map[string]string `mapstructure:"driver_opts"`
	External   External
	Labels     map[string]string `compose:"list_or_dict_equals"`
}

// External identifies a Volume or Network as a reference to a resource that is
// not managed, and should already exist.
type External struct {
	Name     string
	External bool
}
