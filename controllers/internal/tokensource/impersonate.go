package tokensource

import (
	"context"
	"fmt"

	"cloud.google.com/go/iam/credentials/apiv1"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	credentialspb "google.golang.org/genproto/googleapis/iam/credentials/v1"
)

const cloudPlatformScope = "https://www.googleapis.com/auth/cloud-platform"

type impersonateTokenSource struct {
	ctx               context.Context
	sourceTokenSource oauth2.TokenSource
	target            string
	scopes            []string
}

func ImpersonateTokenSource(ctx context.Context, target string, ts oauth2.TokenSource, scopes ...string) (oauth2.TokenSource, error) {
	if len(scopes) == 0 {
		scopes = []string{cloudPlatformScope}
	}
	return &impersonateTokenSource{
		ctx:               ctx,
		sourceTokenSource: ts,
		target:            target,
		scopes:            scopes,
	}, nil
}

func (ts *impersonateTokenSource) Token() (*oauth2.Token, error) {
	client, err := credentials.NewIamCredentialsClient(ts.ctx, option.WithTokenSource(ts.sourceTokenSource))
	if err != nil {
		return nil, fmt.Errorf("iamcredentials.NewIamCredentialsClient: %w", err)
	}
	defer func() { _ = client.Close() }()

	resp, err := client.GenerateAccessToken(ts.ctx, &credentialspb.GenerateAccessTokenRequest{
		Name:  ts.target,
		Scope: ts.scopes,
	})
	if err != nil {
		return nil, fmt.Errorf("iamcredentials.GenerateAccessToken: %w", err)
	}
	return &oauth2.Token{AccessToken: resp.GetAccessToken(), Expiry: resp.GetExpireTime().AsTime()}, nil
}
