// Copyright © 2024 OpenIM open source community. All rights reserved.
package mongoutil

import (
	"context"
	"errors"
	"fmt"
	"net"
	"syscall"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/address"
	"go.mongodb.org/mongo-driver/mongo/description"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
	"go.mongodb.org/mongo-driver/x/mongo/driver/auth"
	"go.mongodb.org/mongo-driver/x/mongo/driver/topology"
)

type failingAuthenticator struct{ err error }

func (a failingAuthenticator) Auth(context.Context, *auth.Config) error { return a.err }

func (a failingAuthenticator) Reauth(context.Context, *auth.Config) error { return a.err }

func (a failingAuthenticator) CreateSpeculativeConversation() (auth.SpeculativeConversation, error) {
	return nil, a.err
}

func handshakeAuthError(err error) error {
	handshaker := auth.Handshaker(nil, &auth.HandshakeOptions{Authenticator: failingAuthenticator{err: err}})
	_, authErr := handshaker.GetHandshakeInformation(context.Background(), address.Address("localhost:27017"), nil)
	return topology.ConnectionError{Wrapped: authErr}
}

func TestShouldRetry(t *testing.T) {
	_, configErr := auth.CreateAuthenticator("unknown", nil, nil)
	if configErr == nil {
		t.Fatal("expected unknown authenticator error")
	}
	wrappedAuth := handshakeAuthError(driver.Error{Code: 18})
	networkErr := &net.OpError{Op: "read", Net: "tcp", Err: syscall.ECONNRESET}
	serverSelection := topology.ServerSelectionError{
		Desc:    description.Topology{Servers: []description.Server{{}, {LastError: networkErr}, {LastError: wrappedAuth}}},
		Wrapped: topology.ErrServerSelectionTimeout,
	}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"command unauthorized", mongo.CommandError{Code: 13}, false},
		{"command auth failed", mongo.CommandError{Code: 18}, false},
		{"wrapped command unauthorized", fmt.Errorf("wrapped: %w", mongo.CommandError{Code: 13}), false},
		{"wrapped command auth failed", fmt.Errorf("wrapped: %w", mongo.CommandError{Code: 18}), false},
		{"wrapped driver unauthorized", fmt.Errorf("wrapped: %w", driver.Error{Code: 13}), false},
		{"handshake auth failed", wrappedAuth, false},
		{"auth configuration", configErr, false},
		{"server selection handshake auth failed", serverSelection, false},
		{"wrapped server selection auth failed", fmt.Errorf("wrapped: %w", serverSelection), false},
		{"transient connection reset", topology.ConnectionError{Wrapped: networkErr}, true},
		{"transient auth handshake reset", handshakeAuthError(networkErr), true},
		{"transient auth handshake timeout", handshakeAuthError(&net.DNSError{IsTimeout: true}), true},
		{"transient server selection", topology.ServerSelectionError{Desc: description.Topology{Servers: []description.Server{{LastError: handshakeAuthError(networkErr)}}}}, true},
		{"shutdown in progress", mongo.CommandError{Code: 91}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldRetry(context.Background(), tt.err); got != tt.want {
				t.Fatalf("shouldRetry() = %v, want %v", got, tt.want)
			}
		})
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if shouldRetry(ctx, errors.New("retry")) {
		t.Fatal("shouldRetry() = true for cancelled context")
	}
}

func TestRetryDelay(t *testing.T) {
	for attempt, delay := range []time.Duration{500 * time.Millisecond, time.Second, 2 * time.Second, 4 * time.Second, 5 * time.Second, 5 * time.Second} {
		for range 100 {
			got := retryDelay(attempt)
			if got < delay/2 || got >= delay {
				t.Fatalf("retryDelay(%d) = %v, want [%v, %v)", attempt, got, delay/2, delay)
			}
		}
	}
}

func TestValidateAndSetDefaults(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		check  func(*testing.T, Config)
	}{
		{
			name:   "standalone defaults and URI",
			config: Config{Database: "app", Address: []string{"localhost:27017"}},
			check: func(t *testing.T, config Config) {
				t.Helper()
				if config.MongoMode != StandaloneMode || config.MaxRetry != defaultMaxRetry || config.MaxPoolSize != defaultMaxPoolSize {
					t.Fatalf("unexpected defaults: %+v", config)
				}
				if config.Uri != buildMongoURI(&config, config.Database) {
					t.Fatalf("URI = %q, want URI built for database %q", config.Uri, config.Database)
				}
			},
		},
		{
			name: "replica set defaults",
			config: Config{
				MongoMode: ReplicaSetMode,
				Database:  "app",
				ReplicaSet: &ReplicaSetConfig{
					Name:  "rs0",
					Hosts: []string{"localhost:27017"},
				},
			},
			check: func(t *testing.T, config Config) {
				t.Helper()
				if config.MaxRetry != defaultMaxRetry || config.MaxPoolSize != defaultMaxPoolSize {
					t.Fatalf("unexpected defaults: %+v", config)
				}
				if config.Database != "app" {
					t.Fatalf("Database = %q, want app", config.Database)
				}
			},
		},
		{
			name:   "replica set requires configuration",
			config: Config{MongoMode: ReplicaSetMode, Database: "app"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.ValidateAndSetDefaults()
			if tt.check == nil {
				if err == nil {
					t.Fatal("ValidateAndSetDefaults() = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ValidateAndSetDefaults() error = %v", err)
			}
			tt.check(t, tt.config)
		})
	}
}
