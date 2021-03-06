/*
Copyright 2021 apstndb.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/apstndb/image-pull-secret-controller/controllers/internal/tokensource"
	"golang.org/x/oauth2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	examplev1alpha1 "github.com/apstndb/image-pull-secret-controller/api/v1alpha1"
)

const (
	cloudPlatformScope = "https://www.googleapis.com/auth/cloud-platform"
	userInfoEmailScope = "https://www.googleapis.com/auth/userinfo.email"
)

// ImagePullSecretReconciler reconciles a ImagePullSecret object
type ImagePullSecretReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	ClientSet *kubernetes.Clientset
}

//+kubebuilder:rbac:groups=core,resources=serviceaccounts/token,verbs=create
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=create;patch;update
//+kubebuilder:rbac:groups=example.apstn.dev,resources=imagepullsecrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=example.apstn.dev,resources=imagepullsecrets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=example.apstn.dev,resources=imagepullsecrets/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ImagePullSecret object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *ImagePullSecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)

	// your logic here
	var imagePullSecret examplev1alpha1.ImagePullSecret
	if err := r.Get(ctx, req.NamespacedName, &imagePullSecret); err != nil {
		return ctrl.Result{RequeueAfter: 10 * time.Second}, err
	}

	reqb, _ := json.Marshal(req)
	resb, _ := json.Marshal(imagePullSecret)
	l.Info("Reconcile:", "reqb", string(reqb), "resource", string(resb))

	if err := r.do(ctx, &imagePullSecret); err != nil {
		l.Error(err, "r.do() failed")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

func (r *ImagePullSecretReconciler) tokenSource(ctx context.Context, res *examplev1alpha1.ImagePullSecret) (oauth2.TokenSource, error) {
	gsaEmail := res.Spec.GsaEmail
	serviceAccountName := res.Spec.ServiceAccountName
	serviceAccountNamespace := res.Namespace
	workloadIdentityPoolProvider := fmt.Sprintf("//iam.googleapis.com/%s", res.Spec.WorkloadIdentityPoolProvider)

	kts, err := tokensource.KubernetesTokenRequestTokenSource(ctx, r.ClientSet,
		&tokensource.KubernetsTokenRequestTokenConfig{
			ServiceAccountNamespace: serviceAccountNamespace,
			ServiceAccountName:      serviceAccountName,
			Audiences:               []string{workloadIdentityPoolProvider},
		})
	if err != nil {
		return nil, err
	}

	stsTs, err := tokensource.OidcStsTokenSource(ctx, workloadIdentityPoolProvider, kts)
	if err != nil {
		return nil, err
	}


	scopes := []string{cloudPlatformScope, userInfoEmailScope}
	impTs, err := tokensource.ImpersonateTokenSource(ctx, gsaEmail, stsTs, scopes...)
	if err != nil {
		return nil, err
	}


	// Wrap to avoid to issue token repeatedly for tokenInfo call
	return oauth2.ReuseTokenSource(nil, impTs), nil
}

func (r *ImagePullSecretReconciler) do(ctx context.Context, res *examplev1alpha1.ImagePullSecret) error {
	ts, err := r.tokenSource(ctx, res)
	if err != nil {
		return err
	}

	// Print token information
	tokeninfoResp, err := tokenInfo(ctx, ts)
	if err != nil {
		return err
	}
	_ = json.NewEncoder(os.Stderr).Encode(tokeninfoResp)

	t, err := ts.Token()
	if err != nil {
		return err
	}

	err = r.upsertDockerConfigSecret(ctx, res, t)
	if err != nil {
		return err
	}

	// Update status only if succeed
	res.Status.ExpiresAt = metav1.NewTime(t.Expiry)
	return r.Status().Update(ctx, res)
}

func (r *ImagePullSecretReconciler) upsertDockerConfigSecret(ctx context.Context, res *examplev1alpha1.ImagePullSecret, token *oauth2.Token) error {
	gsaEmail := res.Spec.GsaEmail
	secretNamespace := res.Namespace
	secretName := res.Spec.SecretName

	b, err := generateDockerConfigJson(gsaEmail, token.AccessToken)
	if err != nil {
		return err
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: secretNamespace,
			Name:      secretName,
		},
		Data: map[string][]byte{".dockerconfigjson": b},
		Type: corev1.SecretTypeDockerConfigJson,
	}
	err = r.Create(ctx, secret)
	if errors.IsAlreadyExists(err) {
		err = r.Update(ctx, secret)
	}
	return err
}

// SetupWithManager sets up the controller with the Manager.
func (r *ImagePullSecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&examplev1alpha1.ImagePullSecret{}).
		Complete(r)
}
