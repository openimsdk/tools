package component

type Mongo struct {
	Address     []string
	Database    string
	Username    string
	Password    string
	MaxPoolSize int
}

type Minio struct {
	Enable          string
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

	LatestMsgToRedis struct {
		Topic string `yaml:"topic"`
	} `yaml:"latestMsgToRedis"`
	MsgToMongo struct {
		Topic string `yaml:"topic"`
	} `yaml:"offlineMsgToMongo"`
	MsgToPush struct {
		Topic string `yaml:"topic"`
	} `yaml:"msgToPush"`
}
