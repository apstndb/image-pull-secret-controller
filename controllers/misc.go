package controllers

import (
	"context"
	"encoding/json"
	"fmt"

	"golang.org/x/oauth2"
	goauth2 "google.golang.org/api/oauth2/v1"
	"google.golang.org/api/option"
)

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

// All GCR hostnames
// https://cloud.google.com/container-registry/docs/overview?hl=en
var gcrRegistries = []string{"gcr.io", "asia.gcr.io", "eu.gcr.io", "us.gcr.io"}

func generateDockerConfigJson(gsaEmail string, accessToken string) ([]byte, error) {
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

func tokenInfo(ctx context.Context, ts oauth2.TokenSource) (*goauth2.Tokeninfo, error) {
	goauth2Svc, err := goauth2.NewService(ctx, option.WithTokenSource(ts))
	if err != nil {
		return nil, fmt.Errorf("oauth2.TokenInfo: %w", err)
	}
	tokeninfo, err := goauth2Svc.Tokeninfo().Do()
	if err != nil {
		return nil, fmt.Errorf("oauth2.TokenInfo: %w", err)
	}
	return tokeninfo, nil
}
