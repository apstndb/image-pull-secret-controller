## Example

Example in `./terraform` directory makes two project.
- Cluster project
  - GKE cluster
- Service project
  - Artifact Registry repo
  - Workload Identity pool & provider
  - Google Service Account(target of impersonate)

### Set up the example using Terraform

Edit and uncomment `billing_account_id` in `terraform/.tfvars`.

```
# Must set billing account ID
# billing_account_id = "012345-6789AB-CDEF01"
```

```
# Set up GCP.
$ (cd terraform/stage1 && terraform apply -var-file ../.tfvars)
# Do `gcloud container clusters get-credentials` for the cluster.
$ (cd terraform/stage1 && terraform output -raw get_credentials_cmd | sh)
# Install the controller.
$ make install deploy
# Apply Kubernetes resources.
$ (cd terraform/stage2 && terraform apply -var-file ../.tfvars)
```

### ImagePullSecret resource

Terraform stage2 applies ImagePullSecret resource like below.

```
apiVersion: example.apstn.dev/v1alpha1
kind: ImagePullSecret
metadata:
  name: imagepullsecret-sample
  namespace: default
spec:
  gsaEmail: image-puller@yourname-example-service-cba2.iam.gserviceaccount.com
  secretName: image-pull-secret
  serviceAccountName: default
  workloadIdentityPoolProvider: projects/628134195223/locations/global/workloadIdentityPools/pool-for-gke/providers/provider-for-gke
```

Example of `kubectl get imagepullsecrets`

```
$ kubectl get imagepullsecrets.example.apstn.dev
NAME                     SECRET              KSA_NAME   GSA_EMAIL                                                            PROVIDER                                                                                               CURRENT_EXPIRES_AT
imagepullsecret-sample   image-pull-secret   default    image-puller@yourname-example-service-cba2.iam.gserviceaccount.com   projects/628134195223/locations/global/workloadIdentityPools/pool-for-gke/providers/provider-for-gke   2021-06-06T16:56:03Z
```

Corresponding secret will be created.

```
$ kubectl get secret image-pull-secret
NAME                  TYPE                                  DATA   AGE
image-pull-secret     kubernetes.io/dockerconfigjson        1      83s
```

### Use the secret

```
$ IMAGE="$(cd terraform/stage1 && terraform output -raw repo)/nginx"                                                           
$ echo $IMAGE
us-central1-docker.pkg.dev/yourname-example-service-cba2/repo/nginx
$ docker pull nginx
$ docker tag nginx ${IMAGE}
$ docker push ${IMAGE}

$ kubectl create deployment --image ${IMAGE} --replicas 1 nginx       
deployment.apps/nginx created

$ kubectl get pod
NAME                     READY   STATUS         RESTARTS   AGE
nginx-6dd8f5ff76-qxsxn   0/1     ErrImagePull   0          14s

$ kubectl patch deployments nginx --patch '{"spec": {"template": {"spec": {"imagePullSecrets": [{"name": "image-pull-secret"}]}}}}'
deployment.apps/nginx patched

$ kubectl get pod     
NAME                    READY   STATUS    RESTARTS   AGE
nginx-f478cb6fb-chsxg   1/1     Running   0          2m9s
```

### Tear down
```
$ (cd terraform/stage2 && terraform destroy) 
$ (cd terraform/stage1 && terraform destroy -var-file ../.tfvars) 
```
