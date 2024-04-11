package main

import (
	"errors"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/mq"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func setupRabbitMQ(ctx *pulumi.Context) error {
	rabbitmqUsername, uerr := ctx.GetConfig("rabbitmq:username")
	rabbitmqPassword, perr := ctx.GetConfig("rabbitmq:password")
	if !uerr || !perr {
		return errors.New("missing required configuration (username/password) parameters")
	}
	broker, err := mq.NewBroker(ctx, "OorbitBroker", &mq.BrokerArgs{
		BrokerName:              pulumi.String("OorbitBroker"),
		EngineType:              pulumi.String("RabbitMQ"),
		EngineVersion:           pulumi.String("3.11.28"),
		HostInstanceType:        pulumi.String("mq.m5.large"),
		PubliclyAccessible:      pulumi.BoolPtr(true),
		AutoMinorVersionUpgrade: pulumi.BoolPtr(true),
		DeploymentMode:          pulumi.String("SINGLE_INSTANCE"),
		Users: mq.BrokerUserArray{
			&mq.BrokerUserArgs{
				Username: pulumi.String(rabbitmqUsername),
				Password: pulumi.String(rabbitmqPassword),
			},
		},
	})
	if err != nil {
		return err
	}
	ctx.Export("brokerName", broker.BrokerName)
	return nil
}
