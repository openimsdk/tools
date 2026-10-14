package kafka

import (
	"strings"
	"testing"

	"github.com/IBM/sarama"
)

func TestBuildConfigSASL(t *testing.T) {
	tests := []struct {
		name              string
		config            Config
		saslEnabled       bool
		mechanism         sarama.SASLMechanism
		hasSCRAMGenerator bool
	}{
		{
			name: "disabled",
		},
		{
			name: "legacy plain",
			config: Config{
				Username: "user",
				Password: "password",
			},
			saslEnabled: true,
			mechanism:   sarama.SASLTypePlaintext,
		},
		{
			name: "explicit plain",
			config: Config{
				Username:      "user",
				Password:      "password",
				SASLMechanism: "plain",
			},
			saslEnabled: true,
			mechanism:   sarama.SASLTypePlaintext,
		},
		{
			name: "scram sha 256",
			config: Config{
				Username:      "user",
				Password:      "password",
				SASLMechanism: "SCRAM-SHA-256",
			},
			saslEnabled:       true,
			mechanism:         sarama.SASLTypeSCRAMSHA256,
			hasSCRAMGenerator: true,
		},
		{
			name: "scram sha 512",
			config: Config{
				Username:      "user",
				Password:      "password",
				SASLMechanism: "scram-sha-512",
			},
			saslEnabled:       true,
			mechanism:         sarama.SASLTypeSCRAMSHA512,
			hasSCRAMGenerator: true,
		},
	}

	builders := []struct {
		name  string
		build func(config Config) (*sarama.Config, error)
	}{
		{
			name:  "producer",
			build: BuildProducerConfig,
		},
		{
			name: "consumer",
			build: func(config Config) (*sarama.Config, error) {
				return BuildConsumerGroupConfig(&config, sarama.OffsetNewest, false)
			},
		},
	}

	for _, builder := range builders {
		t.Run(builder.name, func(t *testing.T) {
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					config, err := builder.build(tt.config)
					if err != nil {
						t.Fatalf("build config failed: %v", err)
					}
					if config.Net.SASL.Enable != tt.saslEnabled {
						t.Fatalf("unexpected SASL enabled state: %v", config.Net.SASL.Enable)
					}
					if !tt.saslEnabled {
						return
					}
					if config.Net.SASL.Mechanism != tt.mechanism {
						t.Fatalf("unexpected SASL mechanism: %s", config.Net.SASL.Mechanism)
					}
					if config.Net.SASL.User != tt.config.Username {
						t.Fatalf("unexpected SASL username: %s", config.Net.SASL.User)
					}
					if config.Net.SASL.Password != tt.config.Password {
						t.Fatal("unexpected SASL password")
					}
					if (config.Net.SASL.SCRAMClientGeneratorFunc != nil) != tt.hasSCRAMGenerator {
						t.Fatal("unexpected SCRAM generator state")
					}
					if tt.hasSCRAMGenerator {
						client := config.Net.SASL.SCRAMClientGeneratorFunc()
						if err := client.Begin(tt.config.Username, tt.config.Password, ""); err != nil {
							t.Fatalf("initialize SCRAM client failed: %v", err)
						}
					}
				})
			}
		})
	}
}

func TestBuildConfigSASLErrors(t *testing.T) {
	tests := []struct {
		name       string
		config     Config
		errMessage string
	}{
		{
			name: "unsupported mechanism",
			config: Config{
				Username:      "user",
				Password:      "password",
				SASLMechanism: "GSSAPI",
			},
			errMessage: "unsupported kafka SASL mechanism",
		},
		{
			name: "scram username required",
			config: Config{
				Password:      "password",
				SASLMechanism: "SCRAM-SHA-512",
			},
			errMessage: "kafka SASL username is required",
		},
		{
			name: "scram password required",
			config: Config{
				Username:      "user",
				SASLMechanism: "SCRAM-SHA-512",
			},
			errMessage: "kafka SASL password is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := BuildProducerConfig(tt.config)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.errMessage) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestSCRAMClientBeginWrapsError(t *testing.T) {
	client := newSCRAMClientGenerator(sarama.SASLTypeSCRAMSHA512)()
	err := client.Begin("\a", "password", "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "initialize kafka SCRAM client failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSCRAMClientBeginDoesNotExposePassword(t *testing.T) {
	const passwordMarker = "secret-password"

	client := newSCRAMClientGenerator(sarama.SASLTypeSCRAMSHA512)()
	err := client.Begin("user", passwordMarker+"\a", "")
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), passwordMarker) {
		t.Fatalf("password leaked in error: %v", err)
	}
}

func TestSCRAMClientStepWrapsError(t *testing.T) {
	client := newSCRAMClientGenerator(sarama.SASLTypeSCRAMSHA512)()
	if err := client.Begin("user", "password", ""); err != nil {
		t.Fatalf("initialize SCRAM client failed: %v", err)
	}
	if _, err := client.Step(""); err != nil {
		t.Fatalf("create SCRAM client first message failed: %v", err)
	}
	_, err := client.Step("invalid-server-first-message")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "advance kafka SCRAM client failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClientCreationErrorsDoNotExposePassword(t *testing.T) {
	const password = "secret-password-must-not-leak"

	tests := []struct {
		name   string
		create func() error
	}{
		{
			name: "producer",
			create: func() error {
				config := sarama.NewConfig()
				config.Net.SASL.Password = password
				_, err := NewProducer(config, nil)
				return err
			},
		},
		{
			name: "consumer group",
			create: func() error {
				config := sarama.NewConfig()
				config.Net.SASL.Password = password
				config.Consumer.Group.Session.Timeout = 0
				_, err := NewConsumerGroup(config, nil, "group")
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.create()
			if err == nil {
				t.Fatal("expected error")
			}
			if strings.Contains(err.Error(), password) {
				t.Fatalf("password leaked in error: %v", err)
			}
		})
	}
}
