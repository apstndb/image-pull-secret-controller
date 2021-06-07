package tokensource

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	"google.golang.org/api/sts/v1"
)

type oidcStsTokenSource struct {
	audience          string
	SourceTokenSource oauth2.TokenSource
	ctx               context.Context
}

func (ts *oidcStsTokenSource) Token() (*oauth2.Token, error) {
	stsSvc, err := sts.NewService(ts.ctx, option.WithoutAuthentication())
	if err != nil {
		return nil, err
	}
	t, err := ts.SourceTokenSource.Token()
	if err != nil {
		return nil, err
	}

	req := &sts.GoogleIdentityStsV1ExchangeTokenRequest{
		Audience:           ts.audience,
		GrantType:          "urn:ietf:params:oauth:grant-type:token-exchange",
		RequestedTokenType: "urn:ietf:params:oauth:token-type:access_token",
		Scope:              "https://www.googleapis.com/auth/iam",
		SubjectToken:       t.AccessToken,
		SubjectTokenType:   "urn:ietf:params:oauth:token-type:jwt",
	}
	if debug {
		_ = json.NewEncoder(os.Stderr).Encode(req)
	}

	// Store base time of ExpiresIn
	now := time.Now()

	resp, err := stsSvc.V1.Token(req).Do()
	if err != nil {
		return nil, fmt.Errorf("sts.Token: %w", err)
	}

	// Citation of GoogleIdentityStsV1ExchangeTokenResponse.ExpiresIn:
	// This field is absent when the `subject_token` in the request
	// is a Google-issued, short-lived access token. In this case, the
	// access token has the same expiration time as the `subject_token`.
	var expiry time.Time
	if resp.ExpiresIn != 0 {
		expiry = now.Add(time.Duration(resp.ExpiresIn) * time.Second)
	} else {
		expiry = t.Expiry
	}
	return &oauth2.Token{AccessToken: resp.AccessToken, Expiry: expiry}, nil
}

// OidcStsTokenSource exchanges OIDC token with federated token for internal use.
func OidcStsTokenSource(ctx context.Context, audience string, ts oauth2.TokenSource) (oauth2.TokenSource, error) {
	return &oidcStsTokenSource{
		ctx:               ctx,
		audience:          audience,
		SourceTokenSource: ts,
	}, nil
}
