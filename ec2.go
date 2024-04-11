package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ebs"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi-tls/sdk/v4/go/tls"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func setupEC2SharedDrive(ctx *pulumi.Context) error {
	filterMostRecent := new(bool)
	*filterMostRecent = true
	windowsAmi, err := ec2.LookupAmi(ctx, &ec2.LookupAmiArgs{
		Filters: []ec2.GetAmiFilter{
			{
				Name:   "name",
				Values: []string{"Windows_Server-2022-English-Full-Base-*"},
			},
		},
		MostRecent: filterMostRecent,
	})
	if err != nil {
		return err
	}
	vpc, err := ec2.NewVpc(ctx, "SharedDriveVPC", &ec2.VpcArgs{
		CidrBlock: pulumi.String("10.0.0.0/16"),
	})
	if err != nil {
		return err
	}

	// Create an Internet Gateway
	gw, err := ec2.NewInternetGateway(ctx, "SharedDriveIGW", &ec2.InternetGatewayArgs{
		VpcId: vpc.ID(),
	}, pulumi.DependsOn([]pulumi.Resource{vpc}))
	if err != nil {
		return err
	}

	// Create a route table that points to the internet gateway
	rt, err := ec2.NewRouteTable(ctx, "SharedDriveRT", &ec2.RouteTableArgs{
		VpcId: vpc.ID(),
		Routes: ec2.RouteTableRouteArray{
			&ec2.RouteTableRouteArgs{
				CidrBlock: pulumi.String("0.0.0.0/0"),
				GatewayId: gw.ID(),
			},
		},
	}, pulumi.DependsOn([]pulumi.Resource{vpc, gw}))
	if err != nil {
		return err
	}

	// Create a subnet
	subnet, err := ec2.NewSubnet(ctx, "SharedDriveSubnet", &ec2.SubnetArgs{
		VpcId:               vpc.ID(),
		CidrBlock:           pulumi.String("10.0.1.0/24"),
		MapPublicIpOnLaunch: pulumi.Bool(true),
	}, pulumi.DependsOn([]pulumi.Resource{vpc}))
	if err != nil {
		return err
	}

	// Associate the subnet with the route table
	_, err = ec2.NewRouteTableAssociation(ctx, "SharedDriveRouteAS", &ec2.RouteTableAssociationArgs{
		RouteTableId: rt.ID(),
		SubnetId:     subnet.ID(),
	}, pulumi.DependsOn([]pulumi.Resource{rt, subnet}))
	if err != nil {
		return err
	}

	// Create a security group
	sg, err := ec2.NewSecurityGroup(ctx, "SharedDriveSecurityGroup", &ec2.SecurityGroupArgs{
		VpcId: vpc.ID(),
		Ingress: ec2.SecurityGroupIngressArray{
			&ec2.SecurityGroupIngressArgs{
				Protocol:   pulumi.String("tcp"),
				FromPort:   pulumi.Int(445), // SMB
				ToPort:     pulumi.Int(445),
				CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
			&ec2.SecurityGroupIngressArgs{
				Protocol:   pulumi.String("tcp"),
				FromPort:   pulumi.Int(139), // SMB
				ToPort:     pulumi.Int(139),
				CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
		},
		Egress: ec2.SecurityGroupEgressArray{
			&ec2.SecurityGroupEgressArgs{ // Allow all outbound traffic.
				Protocol:   pulumi.String("-1"),
				FromPort:   pulumi.Int(0),
				ToPort:     pulumi.Int(0),
				CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
		},
	})
	if err != nil {
		return err
	}

	// Create an EBS volume
	volume, err := ebs.NewVolume(ctx, "SharedDriveVolume", &ebs.VolumeArgs{
		AvailabilityZone: subnet.AvailabilityZone,
		Size:             pulumi.Int(1024),
	})
	if err != nil {
		return err
	}
	// Create new private key
	privateKey, err := tls.NewPrivateKey(ctx, "SharedDrivePrivateKey", &tls.PrivateKeyArgs{
		Algorithm: pulumi.String("RSA"),
		RsaBits:   pulumi.Int(4096),
	})
	if err != nil {
		return err
	}
	// Create a key pair
	key, err := ec2.NewKeyPair(ctx, "SharedDriveKeyPair", &ec2.KeyPairArgs{
		KeyName:   pulumi.String("shared-drive-key"),
		PublicKey: privateKey.PublicKeyOpenssh,
	}, pulumi.Protect(true))
	if err != nil {
		return err
	}
	// Create an EC2 instance
	instance, err := ec2.NewInstance(ctx, "SharedDriveInstance", &ec2.InstanceArgs{
		Ami:                      pulumi.String(windowsAmi.Id),
		InstanceType:             pulumi.String("m5dn.2xlarge"),
		KeyName:                  key.KeyName,
		SubnetId:                 subnet.ID(),
		VpcSecurityGroupIds:      pulumi.StringArray{sg.ID()},
		AssociatePublicIpAddress: pulumi.Bool(true),
	}, pulumi.DependsOn([]pulumi.Resource{subnet, sg, key}))
	if err != nil {
		return err
	}

	// Shared Drive
	_, err = ec2.NewVolumeAttachment(ctx, "SharedDriveAttachment", &ec2.VolumeAttachmentArgs{
		DeviceName: pulumi.String("/dev/sdf"),
		InstanceId: instance.ID(),
		VolumeId:   volume.ID(),
	}, pulumi.DependsOn([]pulumi.Resource{instance, volume}))
	if err != nil {
		return err
	}

	// Exports
	ctx.Export("instanceId", instance.ID())
	ctx.Export("instancePublicIP", instance.PublicIp)
	ctx.Export("privateKey", privateKey.PrivateKeyPem)
	ctx.Export("publicKey", privateKey.PublicKeyOpenssh)
	return nil
}
