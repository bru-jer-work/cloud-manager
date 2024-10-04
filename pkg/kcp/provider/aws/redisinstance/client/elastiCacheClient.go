package client

import (
	"context"
	"fmt"
	"math"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"

	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	elasticache "github.com/aws/aws-sdk-go-v2/service/elasticache"
	elasticacheTypes "github.com/aws/aws-sdk-go-v2/service/elasticache/types"
	secretsmanager "github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	secretsmanagerTypes "github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/google/uuid"
	awsclient "github.com/kyma-project/cloud-manager/pkg/kcp/provider/aws/client"
	awsmeta "github.com/kyma-project/cloud-manager/pkg/kcp/provider/aws/meta"

	"k8s.io/utils/ptr"
)

func NewClientProvider() awsclient.SkrClientProvider[ElastiCacheClient] {
	return func(ctx context.Context, account, region, key, secret, role string) (ElastiCacheClient, error) {
		cfg, err := awsclient.NewSkrConfig(ctx, region, key, secret, role)
		if err != nil {
			return nil, err
		}

		return newClient(
			ec2.NewFromConfig(cfg),
			elasticache.NewFromConfig(cfg),
			secretsmanager.NewFromConfig(cfg),
		), nil
	}
}

type CreateElastiCacheClusterOptions struct {
	Name                       string
	SubnetGroupName            string
	ParameterGroupName         string
	CacheNodeType              string
	EngineVersion              string
	AutoMinorVersionUpgrade    bool
	AuthTokenSecretString      *string
	TransitEncryptionEnabled   bool
	PreferredMaintenanceWindow *string
	SecurityGroupIds           []string
}

type ModifyElastiCacheClusterOptions struct {
	CacheNodeType              *string
	EngineVersion              *string
	AutoMinorVersionUpgrade    *bool
	PreferredMaintenanceWindow *string
	TransitEncryptionEnabled   *bool
	TransitEncryptionMode      *elasticacheTypes.TransitEncryptionMode
	AuthTokenSecretString      *string
	UserGroupIdsToAdd          []string
	UserGroupIdsToRemove       []string
}

type ElastiCacheClient interface {
	DescribeElastiCacheSubnetGroup(ctx context.Context, name string) ([]elasticacheTypes.CacheSubnetGroup, error)
	CreateElastiCacheSubnetGroup(ctx context.Context, name string, subnetIds []string, tags []elasticacheTypes.Tag) (*elasticache.CreateCacheSubnetGroupOutput, error)
	DeleteElastiCacheSubnetGroup(ctx context.Context, name string) error

	DescribeElastiCacheParameterGroup(ctx context.Context, name string) ([]elasticacheTypes.CacheParameterGroup, error)
	CreateElastiCacheParameterGroup(ctx context.Context, name, family string, tags []elasticacheTypes.Tag) (*elasticache.CreateCacheParameterGroupOutput, error)
	DeleteElastiCacheParameterGroup(ctx context.Context, name string) error
	DescribeElastiCacheParameters(ctx context.Context, groupName string) ([]elasticacheTypes.Parameter, error)
	ModifyElastiCacheParameterGroup(ctx context.Context, groupName string, parameters []elasticacheTypes.ParameterNameValue) error
	DescribeEngineDefaultParameters(ctx context.Context, family string) ([]elasticacheTypes.Parameter, error)

	GetAuthTokenSecretValue(ctx context.Context, secretName string) (*secretsmanager.GetSecretValueOutput, error)
	CreateAuthTokenSecret(ctx context.Context, secretName string, tags []secretsmanagerTypes.Tag) error
	DeleteAuthTokenSecret(ctx context.Context, secretName string) error

	DescribeElastiCacheReplicationGroup(ctx context.Context, clusterId string) ([]elasticacheTypes.ReplicationGroup, error)
	CreateElastiCacheReplicationGroup(ctx context.Context, tags []elasticacheTypes.Tag, options CreateElastiCacheClusterOptions) (*elasticache.CreateReplicationGroupOutput, error)
	ModifyElastiCacheReplicationGroup(ctx context.Context, id string, options ModifyElastiCacheClusterOptions) (*elasticache.ModifyReplicationGroupOutput, error)
	DeleteElastiCacheReplicationGroup(ctx context.Context, id string) error
	DescribeElastiCacheCluster(ctx context.Context, id string) ([]elasticacheTypes.CacheCluster, error)

	DescribeUserGroup(ctx context.Context, id string) (*elasticacheTypes.UserGroup, error)
	CreateUserGroup(ctx context.Context, id string, tags []elasticacheTypes.Tag) (*elasticache.CreateUserGroupOutput, error)
	DeleteUserGroup(ctx context.Context, id string) error

	DescribeElastiCacheSecurityGroups(ctx context.Context, filters []ec2Types.Filter, groupIds []string) ([]ec2Types.SecurityGroup, error)
	CreateElastiCacheSecurityGroup(ctx context.Context, vpcId, name string, tags []ec2Types.Tag) (string, error)
	DeleteElastiCacheSecurityGroup(ctx context.Context, id string) error
	AuthorizeElastiCacheSecurityGroupIngress(ctx context.Context, groupId string, ipPermissions []ec2Types.IpPermission) error
}

func newClient(ec2Svc *ec2.Client, elastiCacheSvc *elasticache.Client, secretsManagerSvc *secretsmanager.Client) ElastiCacheClient {
	return &client{
		ec2Svc:            ec2Svc,
		elastiCacheSvc:    elastiCacheSvc,
		secretsManagerSvc: secretsManagerSvc,
	}
}

type client struct {
	ec2Svc            *ec2.Client
	elastiCacheSvc    *elasticache.Client
	secretsManagerSvc *secretsmanager.Client
}

func (c *client) DescribeElastiCacheSubnetGroup(ctx context.Context, name string) ([]elasticacheTypes.CacheSubnetGroup, error) {

	out, err := c.elastiCacheSvc.DescribeCacheSubnetGroups(ctx, &elasticache.DescribeCacheSubnetGroupsInput{
		CacheSubnetGroupName: ptr.To(name),
	})
	if err != nil {
		if awsmeta.IsNotFound(err) {
			return []elasticacheTypes.CacheSubnetGroup{}, nil
		}

		return nil, err
	}

	return out.CacheSubnetGroups, nil
}

func (c *client) CreateElastiCacheSubnetGroup(ctx context.Context, name string, subnetIds []string, tags []elasticacheTypes.Tag) (*elasticache.CreateCacheSubnetGroupOutput, error) {
	out, err := c.elastiCacheSvc.CreateCacheSubnetGroup(ctx, &elasticache.CreateCacheSubnetGroupInput{
		CacheSubnetGroupDescription: ptr.To(fmt.Sprintf("SubnetGroup for ElastiCache %s", name)),
		CacheSubnetGroupName:        ptr.To(name),
		Tags:                        tags,
		SubnetIds:                   subnetIds,
	})

	if err != nil {
		return nil, err
	}

	return out, nil
}

func (c *client) DeleteElastiCacheSubnetGroup(ctx context.Context, name string) error {
	_, err := c.elastiCacheSvc.DeleteCacheSubnetGroup(ctx, &elasticache.DeleteCacheSubnetGroupInput{
		CacheSubnetGroupName: ptr.To(name),
	})

	if err != nil {
		return err
	}

	return nil
}

func (c *client) DescribeElastiCacheParameterGroup(ctx context.Context, name string) ([]elasticacheTypes.CacheParameterGroup, error) {
	out, err := c.elastiCacheSvc.DescribeCacheParameterGroups(ctx, &elasticache.DescribeCacheParameterGroupsInput{
		CacheParameterGroupName: ptr.To(name),
	})
	if err != nil {
		if awsmeta.IsNotFound(err) {
			return []elasticacheTypes.CacheParameterGroup{}, nil
		}

		return nil, err
	}

	return out.CacheParameterGroups, nil
}

func (c *client) CreateElastiCacheParameterGroup(ctx context.Context, name, family string, tags []elasticacheTypes.Tag) (*elasticache.CreateCacheParameterGroupOutput, error) {
	out, err := c.elastiCacheSvc.CreateCacheParameterGroup(ctx, &elasticache.CreateCacheParameterGroupInput{
		CacheParameterGroupName:   ptr.To(name),
		CacheParameterGroupFamily: ptr.To(family),
		Tags:                      tags,
		Description:               ptr.To(fmt.Sprintf("ParameterGroup for ElastiCache %s", name)),
	})

	if err != nil {
		return nil, err
	}

	return out, nil
}

func (c *client) DeleteElastiCacheParameterGroup(ctx context.Context, name string) error {
	_, err := c.elastiCacheSvc.DeleteCacheParameterGroup(ctx, &elasticache.DeleteCacheParameterGroupInput{
		CacheParameterGroupName: ptr.To(name),
	})

	if err != nil {
		return err
	}

	return nil
}

func (c *client) DescribeElastiCacheParameters(ctx context.Context, groupName string) ([]elasticacheTypes.Parameter, error) {
	result := []elasticacheTypes.Parameter{}
	var marker *string = nil
	for {
		out, err := c.elastiCacheSvc.DescribeCacheParameters(ctx, &elasticache.DescribeCacheParametersInput{
			CacheParameterGroupName: ptr.To(groupName),
			Marker:                  marker,
			MaxRecords:              ptr.To(int32(20)),
		})

		if err != nil {
			return nil, err
		}

		result = append(result, out.Parameters...)

		if out.Marker == nil {
			break
		}
		marker = out.Marker
	}

	return result, nil
}

func (c *client) ModifyElastiCacheParameterGroup(ctx context.Context, groupName string, parameters []elasticacheTypes.ParameterNameValue) error {
	apiMaxChunkSize := 20
	for i := 0; i < len(parameters); i += apiMaxChunkSize {
		end := int(math.Min(float64(i+apiMaxChunkSize), float64(len(parameters))))

		chunk := parameters[i:end]

		_, err := c.elastiCacheSvc.ModifyCacheParameterGroup(ctx, &elasticache.ModifyCacheParameterGroupInput{
			CacheParameterGroupName: ptr.To(groupName),
			ParameterNameValues:     chunk,
		})

		if err != nil {
			return err
		}
	}

	return nil
}

var defaultParametersCache map[string][]elasticacheTypes.Parameter = map[string][]elasticacheTypes.Parameter{}

func (c *client) DescribeEngineDefaultParameters(ctx context.Context, family string) ([]elasticacheTypes.Parameter, error) {
	if _, found := defaultParametersCache[family]; found {
		return defaultParametersCache[family], nil
	}

	result := []elasticacheTypes.Parameter{}
	var marker *string = nil
	for {
		out, err := c.elastiCacheSvc.DescribeEngineDefaultParameters(ctx, &elasticache.DescribeEngineDefaultParametersInput{
			CacheParameterGroupFamily: ptr.To(family),
			Marker:                    marker,
		})

		if err != nil {
			return nil, err
		}

		result = append(result, out.EngineDefaults.Parameters...)
		if out.EngineDefaults.Marker == nil {
			break
		}
	}

	defaultParametersCache[family] = result

	return result, nil
}

func (c *client) GetAuthTokenSecretValue(ctx context.Context, secretName string) (*secretsmanager.GetSecretValueOutput, error) {
	out, err := c.secretsManagerSvc.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: ptr.To(secretName),
	})

	if err != nil {
		if awsmeta.IsNotFound(err) {
			return nil, nil
		}

		return nil, err
	}

	return out, nil
}

func (c *client) CreateAuthTokenSecret(ctx context.Context, secretName string, tags []secretsmanagerTypes.Tag) error {
	_, err := c.secretsManagerSvc.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
		Name:         ptr.To(secretName),
		SecretString: ptr.To(uuid.NewString()),
		Tags:         tags,
	})

	return err
}

func (c *client) DeleteAuthTokenSecret(ctx context.Context, secretName string) error {
	_, err := c.secretsManagerSvc.DeleteSecret(ctx, &secretsmanager.DeleteSecretInput{
		SecretId:                   ptr.To(secretName),
		ForceDeleteWithoutRecovery: aws.Bool(true),
	})

	return err
}

func (c *client) DescribeElastiCacheReplicationGroup(ctx context.Context, clusterId string) ([]elasticacheTypes.ReplicationGroup, error) {
	out, err := c.elastiCacheSvc.DescribeReplicationGroups(ctx, &elasticache.DescribeReplicationGroupsInput{
		ReplicationGroupId: ptr.To(clusterId),
	})

	if err != nil {
		if awsmeta.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return out.ReplicationGroups, nil
}

func (c *client) CreateElastiCacheReplicationGroup(ctx context.Context, tags []elasticacheTypes.Tag, options CreateElastiCacheClusterOptions) (*elasticache.CreateReplicationGroupOutput, error) {
	params := &elasticache.CreateReplicationGroupInput{
		ReplicationGroupId:          aws.String(options.Name),
		ReplicationGroupDescription: aws.String("ElastiCache managed by Kyma Cloud Manager"),
		CacheSubnetGroupName:        aws.String(options.SubnetGroupName),
		CacheParameterGroupName:     aws.String(options.ParameterGroupName),
		CacheNodeType:               aws.String(options.CacheNodeType),
		NumCacheClusters:            aws.Int32(1),
		NumNodeGroups:               aws.Int32(1),
		ClusterMode:                 elasticacheTypes.ClusterModeDisabled,
		Engine:                      aws.String("redis"),
		EngineVersion:               aws.String(options.EngineVersion),
		AutoMinorVersionUpgrade:     aws.Bool(options.AutoMinorVersionUpgrade),
		AuthToken:                   options.AuthTokenSecretString,
		TransitEncryptionEnabled:    aws.Bool(options.TransitEncryptionEnabled),
		PreferredMaintenanceWindow:  options.PreferredMaintenanceWindow,
		SecurityGroupIds:            options.SecurityGroupIds,
		Tags:                        tags,
	}
	res, err := c.elastiCacheSvc.CreateReplicationGroup(ctx, params)

	if err != nil {
		return nil, err
	}

	return res, nil
}

func (c *client) ModifyElastiCacheReplicationGroup(ctx context.Context, id string, options ModifyElastiCacheClusterOptions) (*elasticache.ModifyReplicationGroupOutput, error) {
	params := &elasticache.ModifyReplicationGroupInput{
		ReplicationGroupId: aws.String(id),
		ApplyImmediately:   aws.Bool(true),
	}
	if options.CacheNodeType != nil {
		params.CacheNodeType = options.CacheNodeType
	}
	if options.EngineVersion != nil {
		params.EngineVersion = options.EngineVersion
	}
	if options.PreferredMaintenanceWindow != nil {
		params.PreferredMaintenanceWindow = options.PreferredMaintenanceWindow
	}
	if options.AutoMinorVersionUpgrade != nil {
		params.AutoMinorVersionUpgrade = options.AutoMinorVersionUpgrade
	}
	if options.TransitEncryptionEnabled != nil {
		params.TransitEncryptionEnabled = options.TransitEncryptionEnabled
	}
	if options.TransitEncryptionMode != nil {
		params.TransitEncryptionMode = *options.TransitEncryptionMode
	}
	if options.AuthTokenSecretString != nil {
		params.AuthToken = options.AuthTokenSecretString
	}
	if len(options.UserGroupIdsToAdd) > 0 {
		params.UserGroupIdsToAdd = options.UserGroupIdsToAdd
		params.AuthTokenUpdateStrategy = elasticacheTypes.AuthTokenUpdateStrategyTypeDelete
	}
	if len(options.UserGroupIdsToRemove) > 0 {
		params.UserGroupIdsToRemove = options.UserGroupIdsToRemove
	}

	res, err := c.elastiCacheSvc.ModifyReplicationGroup(ctx, params)

	if err != nil {
		return nil, err
	}

	return res, nil
}

func (c *client) DeleteElastiCacheReplicationGroup(ctx context.Context, id string) error {
	deleteInput := &elasticache.DeleteReplicationGroupInput{
		ReplicationGroupId:   ptr.To(id),
		RetainPrimaryCluster: aws.Bool(false),
	}

	_, err := c.elastiCacheSvc.DeleteReplicationGroup(ctx, deleteInput)

	return err
}

func (c *client) DescribeElastiCacheCluster(ctx context.Context, id string) ([]elasticacheTypes.CacheCluster, error) {
	out, err := c.elastiCacheSvc.DescribeCacheClusters(ctx, &elasticache.DescribeCacheClustersInput{
		CacheClusterId: ptr.To(id),
	})

	if err != nil {
		if awsmeta.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return out.CacheClusters, nil
}

func (c *client) DescribeUserGroup(ctx context.Context, id string) (*elasticacheTypes.UserGroup, error) {
	res, err := c.elastiCacheSvc.DescribeUserGroups(ctx, &elasticache.DescribeUserGroupsInput{
		UserGroupId: ptr.To(id),
	})

	if err != nil {
		if awsmeta.IsNotFound(err) {
			return nil, nil
		}

		return nil, err
	}

	if len(res.UserGroups) == 0 {
		return nil, nil
	}

	return ptr.To(res.UserGroups[0]), nil
}

func (c *client) CreateUserGroup(ctx context.Context, id string, tags []elasticacheTypes.Tag) (*elasticache.CreateUserGroupOutput, error) {
	res, err := c.elastiCacheSvc.CreateUserGroup(ctx, &elasticache.CreateUserGroupInput{
		UserGroupId: ptr.To(id),
		Engine:      ptr.To("redis"),
		Tags:        tags,
		UserIds:     []string{"default"},
	})
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (c *client) DeleteUserGroup(ctx context.Context, id string) error {
	_, err := c.elastiCacheSvc.DeleteUserGroup(ctx, &elasticache.DeleteUserGroupInput{
		UserGroupId: ptr.To(id),
	})

	if err != nil {
		return err
	}

	return nil
}

func (c *client) DescribeElastiCacheSecurityGroups(ctx context.Context, filters []ec2Types.Filter, groupIds []string) ([]ec2Types.SecurityGroup, error) {
	out, err := c.ec2Svc.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
		Filters:  filters,
		GroupIds: groupIds,
	})
	if err != nil {
		return nil, err
	}
	return out.SecurityGroups, nil
}

func (c *client) CreateElastiCacheSecurityGroup(ctx context.Context, vpcId, name string, tags []ec2Types.Tag) (string, error) {
	out, err := c.ec2Svc.CreateSecurityGroup(ctx, &ec2.CreateSecurityGroupInput{
		Description: ptr.To(fmt.Sprintf("SecurityGroup for ElastiCache %s", name)),
		GroupName:   ptr.To(name),
		TagSpecifications: []ec2Types.TagSpecification{
			{
				ResourceType: ec2Types.ResourceTypeSecurityGroup,
				Tags:         tags,
			},
		},
		VpcId: ptr.To(vpcId),
	})
	if err != nil {
		return "", err
	}
	return ptr.Deref(out.GroupId, ""), nil
}

func (c *client) DeleteElastiCacheSecurityGroup(ctx context.Context, id string) error {
	in := &ec2.DeleteSecurityGroupInput{
		GroupId: ptr.To(id),
	}
	_, err := c.ec2Svc.DeleteSecurityGroup(ctx, in)
	return err
}

func (c *client) AuthorizeElastiCacheSecurityGroupIngress(ctx context.Context, groupId string, ipPermissions []ec2Types.IpPermission) error {
	_, err := c.ec2Svc.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId:       ptr.To(groupId),
		IpPermissions: ipPermissions,
	})
	if err != nil {
		return err
	}
	return nil
}
