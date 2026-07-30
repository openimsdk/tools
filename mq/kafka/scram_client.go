// Copyright 2026 OpenIM open source community. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kafka

import (
	"github.com/IBM/sarama"
	"github.com/openimsdk/tools/errs"
	"github.com/xdg-go/scram"
)

type scramClient struct {
	hashGenerator scram.HashGeneratorFcn
	conversation  *scram.ClientConversation
}

func newSCRAMClientGenerator(mechanism sarama.SASLMechanism) func() sarama.SCRAMClient {
	hashGenerator := scram.SHA256
	if mechanism == sarama.SASLTypeSCRAMSHA512 {
		hashGenerator = scram.SHA512
	}
	return func() sarama.SCRAMClient {
		return &scramClient{hashGenerator: hashGenerator}
	}
}

func (s *scramClient) Begin(userName, password, authzID string) error {
	client, err := s.hashGenerator.NewClient(userName, password, authzID)
	if err != nil {
		return errs.New("initialize kafka SCRAM client failed").Wrap()
	}
	s.conversation = client.NewConversation()
	return nil
}

func (s *scramClient) Step(challenge string) (string, error) {
	response, err := s.conversation.Step(challenge)
	if err != nil {
		return "", errs.WrapMsg(err, "advance kafka SCRAM client failed")
	}
	return response, nil
}

func (s *scramClient) Done() bool {
	return s.conversation.Done()
}
