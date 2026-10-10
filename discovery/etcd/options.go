package etcd

import (
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// CfgOption defines a function type for modifying clientv3.Config
type CfgOption func(*clientv3.Config)

// WithDialTimeout sets a custom dial timeout for the etcd client
func WithDialTimeout(timeout time.Duration) CfgOption {
	return func(cfg *clientv3.Config) {
		cfg.DialTimeout = timeout
	}
}

// WithMaxCallSendMsgSize sets a custom max call send message size for the etcd client
func WithMaxCallSendMsgSize(size int) CfgOption {
	return func(cfg *clientv3.Config) {
		cfg.MaxCallSendMsgSize = size
	}
}

// WithUsernameAndPassword sets a username and password for the etcd client
func WithUsernameAndPassword(username, password string) CfgOption {
	return func(cfg *clientv3.Config) {
		cfg.Username = username
		cfg.Password = password
	}
}
