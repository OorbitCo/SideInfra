package main

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		err := setupRabbitMQ(ctx)
		if err != nil {
			return err
		}
		err = setupEC2SharedDrive(ctx)
		return err
	})
}
