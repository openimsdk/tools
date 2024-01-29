package component

type Mongo struct {
	URL         string
	Address     []string
	Database    string
	Username    string
	Password    string
	MaxPoolSize int
}

type Minio struct {
	ApiURL          string
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	SignEndpoint    string
	UseSSL          string
}

type Redis struct {
	Address  []string
	Username string
	Password string
}

type Zookeeper struct {
	Schema   string
	ZkAddr   []string
	Username string
	Password string
}

type MySQL struct {
	Address  []string
	Username string
	Password string
	Database string
}

type Kafka struct {
	Username string   `yaml:"username"`
	Password string   `yaml:"password"`
	Addr     []string `yaml:"addr"`
}
