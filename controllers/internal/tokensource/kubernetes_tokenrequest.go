package tokensource

import (
	"context"

	"golang.org/x/oauth2"
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type KubernetsTokenRequestTokenConfig struct {
	ServiceAccountNamespace string
	ServiceAccountName      string
	Audiences               []string
}

type tokenRequestTokenSource struct {
	KubernetsTokenRequestTokenConfig
	ctx       context.Context
	clientset *kubernetes.Clientset
}

func (t tokenRequestTokenSource) Token() (*oauth2.Token, error) {
	tokenRequestResp, err := t.clientset.
		CoreV1().
		ServiceAccounts(t.ServiceAccountNamespace).
		CreateToken(
			t.ctx, t.ServiceAccountName,
			&authenticationv1.TokenRequest{
				Spec: authenticationv1.TokenRequestSpec{
					Audiences: t.Audiences,
				},
			},
			metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return &oauth2.Token{
		AccessToken: tokenRequestResp.Status.Token,
		Expiry:      tokenRequestResp.Status.ExpirationTimestamp.Time,
	}, nil
}

func KubernetesTokenRequestTokenSource(ctx context.Context, clientset *kubernetes.Clientset, config *KubernetsTokenRequestTokenConfig) (oauth2.TokenSource, error) {
	return &tokenRequestTokenSource{
		ctx:                              ctx,
		clientset:                        clientset,
		KubernetsTokenRequestTokenConfig: *config,
	}, nil
}
