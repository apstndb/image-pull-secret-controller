package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	credentials "cloud.google.com/go/iam/credentials/apiv1"
	"golang.org/x/oauth2"
	goauth2 "google.golang.org/api/oauth2/v1"
	"google.golang.org/api/option"
	"google.golang.org/api/sts/v1"
	credentialspb "google.golang.org/genproto/googleapis/iam/credentials/v1"
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var debug bool

func init() {
	debug = os.Getenv("DEBUG") != ""
}

func CreateTokenForAudiences(ctx context.Context, clientset *kubernetes.Clientset, namespace string, serviceAccountName string, audiences []string) (*authenticationv1.TokenRequest, error) {
	tokenRequestResp, err := clientset.CoreV1().ServiceAccounts(namespace).CreateToken(ctx, serviceAccountName, &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			Audiences: audiences,
		},
	}, metav1.CreateOptions{})
	return tokenRequestResp, err
}

func ExchangeJwtForFederatedToken(ctx context.Context, workloadIdentityProvider string, jwt string) (*sts.GoogleIdentityStsV1ExchangeTokenResponse, error) {
	stsSvc, err := sts.NewService(ctx, option.WithoutAuthentication())
	if err != nil {
		return nil, err
	}

	req := &sts.GoogleIdentityStsV1ExchangeTokenRequest{
		Audience:           workloadIdentityProvider,
		GrantType:          "urn:ietf:params:oauth:grant-type:token-exchange",
		RequestedTokenType: "urn:ietf:params:oauth:token-type:access_token",
		Scope:              "https://www.googleapis.com/auth/iam",
		SubjectToken:       jwt,
		SubjectTokenType:   "urn:ietf:params:oauth:token-type:jwt",
	}
	if debug {
		json.NewEncoder(os.Stderr).Encode(req)
	}
	stsTokenResp, err := stsSvc.V1.Token(
		req).Do()
	if err != nil {
		return nil, fmt.Errorf("sts.Token: %w", err)
	}
	return stsTokenResp, nil
}

// gcloud artifacts locations list --format='value(format("\"{0}-docker.pkg.dev\",", name))'
var artifactRegistries = []string{
	"asia-docker.pkg.dev",
	"asia-east1-docker.pkg.dev",
	"asia-east2-docker.pkg.dev",
	"asia-northeast1-docker.pkg.dev",
	"asia-northeast2-docker.pkg.dev",
	"asia-northeast3-docker.pkg.dev",
	"asia-south1-docker.pkg.dev",
	"asia-southeast1-docker.pkg.dev",
	"asia-southeast2-docker.pkg.dev",
	"australia-southeast1-docker.pkg.dev",
	"europe-docker.pkg.dev",
	"europe-north1-docker.pkg.dev",
	"europe-west1-docker.pkg.dev",
	"europe-west2-docker.pkg.dev",
	"europe-west3-docker.pkg.dev",
	"europe-west4-docker.pkg.dev",
	"europe-west6-docker.pkg.dev",
	"northamerica-northeast1-docker.pkg.dev",
	"southamerica-east1-docker.pkg.dev",
	"us-docker.pkg.dev",
	"us-central1-docker.pkg.dev",
	"us-east1-docker.pkg.dev",
	"us-east4-docker.pkg.dev",
	"us-west1-docker.pkg.dev",
	"us-west2-docker.pkg.dev",
	"us-west3-docker.pkg.dev",
	"us-west4-docker.pkg.dev",
}

var gcrRegistries = []string{"gcr.io", "asia.gcr.io", "eu.gcr.io", "us.gcr.io"}

func GenerateDockerConfigJson(gsaEmail string, accessToken string) ([]byte, error) {
	var registries []string
	registries = append(registries, gcrRegistries...)
	registries = append(registries, artifactRegistries...)

	type dockerCfgAuth struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Email    string `json:"email"`
	}
	type dockerCfg struct {
		Auths map[string]dockerCfgAuth `json:"auths"`
	}
	cfg := dockerCfg{Auths: make(map[string]dockerCfgAuth)}
	for _, reg := range registries {
		cfg.Auths[reg] = dockerCfgAuth{
			Username: "oauth2accesstoken",
			Password: accessToken,
			Email:    gsaEmail,
		}
	}
	j, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("json.Marshal: %w", err)
	}
	return j, nil
}

func TokenInfo(ctx context.Context, ts oauth2.TokenSource) (*goauth2.Tokeninfo, error) {
	go2Svc, err := goauth2.NewService(ctx, option.WithTokenSource(ts))
	if err != nil {
		return nil, fmt.Errorf("oauth2.TokenInfo: %w", err)
	}
	tokeninfo, err := go2Svc.Tokeninfo().Do()
	if err != nil {
		return nil, fmt.Errorf("oauth2.TokenInfo: %w", err)
	}
	return tokeninfo, nil
}

func ImpersonateAccessToken(ctx context.Context, gsaEmail string, ts oauth2.TokenSource) (*credentialspb.GenerateAccessTokenResponse, error) {
	iamCredentialsClient, err := credentials.NewIamCredentialsClient(ctx, option.WithTokenSource(ts))
	if err != nil {
		return nil, fmt.Errorf("credentials.NewIamCredentialsClient: %w", err)
	}
	defer iamCredentialsClient.Close()
	accessToken, err := iamCredentialsClient.GenerateAccessToken(ctx, &credentialspb.GenerateAccessTokenRequest{
		Name:  gsaEmail,
		Scope: []string{"https://www.googleapis.com/auth/cloud-platform", "https://www.googleapis.com/auth/userinfo.email"},
	})
	if err != nil {
		return nil, fmt.Errorf("credentials.GenerateAccessToken: %w", err)
	}
	return accessToken, nil
}
