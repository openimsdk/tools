package kafka

import (
	"github.com/IBM/sarama"
	"github.com/openimsdk/tools/errs"
)

func BuildConsumerGroupConfig(conf *Config, initial int64) (*sarama.Config, error) {
	kfk := sarama.NewConfig()
	kfk.Version = sarama.V2_0_0_0
	kfk.Consumer.Offsets.Initial = initial
	kfk.Consumer.Return.Errors = false
	if conf.Username != "" || conf.Password != "" {
		kfk.Net.SASL.Enable = true
		kfk.Net.SASL.User = conf.Username
		kfk.Net.SASL.Password = conf.Password
	}
	if conf.TLS != nil {
		tls, err := newTLSConfig(conf.TLS.ClientCrt, conf.TLS.ClientKey, conf.TLS.CACrt, []byte(conf.TLS.ClientKeyPwd), conf.TLS.InsecureSkipVerify)
		if err != nil {
			return nil, err
		}
		kfk.Net.TLS.Config = tls
		kfk.Net.TLS.Enable = true
	}
	return kfk, nil
}

func NewConsumerGroup(conf *sarama.Config, addr []string, groupID string) (sarama.ConsumerGroup, error) {
	cg, err := sarama.NewConsumerGroup(addr, groupID, conf)
	if err != nil {
		return nil, errs.WrapMsg(err, "NewConsumerGroup failed", "addr", addr, "groupID", groupID, "conf", *conf)
	}
	return cg, nil
}
