package apigee

import (
	"context"
	"fmt"

	apihub "cloud.google.com/go/apihub/apiv1"
	apihubpb "cloud.google.com/go/apihub/apiv1/apihubpb"
	"google.golang.org/api/option"
)

type Client struct {
	apiHubClient  *apihub.ApiHubPluginClient
	collectClient *apihub.ApiHubCollectClient
}

func NewClient(ctx context.Context) (*Client, error) {
	opts := option.WithScopes("https://www.googleapis.com/auth/cloud-platform")
	apiHubClient, err := apihub.NewApiHubPluginRESTClient(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create API Hub client: %v", err)
	}
	collectClient, err := apihub.NewApiHubCollectRESTClient(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create API Hub Collect client: %v", err)
	}
	return &Client{
		apiHubClient:  apiHubClient,
		collectClient: collectClient,
	}, nil
}

func (c *Client) GetPluginInstance(ctx context.Context, name string) (*apihubpb.PluginInstance, error) {
	return c.apiHubClient.GetPluginInstance(ctx, &apihubpb.GetPluginInstanceRequest{Name: name})
}

func (c *Client) CollectApiData(ctx context.Context, parent string, request *apihubpb.CollectApiDataRequest) (*apihubpb.CollectApiDataResponse, error) {
	op, err := c.collectClient.CollectApiData(ctx, request)
	if err != nil {
		return nil, err
	}
	return op.Wait(ctx)
}
