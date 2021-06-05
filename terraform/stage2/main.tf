data "terraform_remote_state" "infrastructure" {
  backend = "local"

  config = {
    path = "../stage1/terraform.tfstate"
  }
}

provider "google" {}

provider "kubernetes" {
  host  = "https://${data.google_container_cluster.cluster.endpoint}"
  token = data.google_client_config.provider.access_token
  cluster_ca_certificate = base64decode(
  data.google_container_cluster.cluster.master_auth[0].cluster_ca_certificate
  )
}

provider "kubernetes-alpha" {
  host  = "https://${data.google_container_cluster.cluster.endpoint}"
  token = data.google_client_config.provider.access_token
  cluster_ca_certificate = base64decode(
    data.google_container_cluster.cluster.master_auth[0].cluster_ca_certificate
  )
}

data "google_client_config" "provider" {
  provider = google
}

data "google_container_cluster" "cluster" {
  provider = google
  project  = data.terraform_remote_state.infrastructure.outputs.cluster_project_id
  name  = data.terraform_remote_state.infrastructure.outputs.cluster_name
  location  = data.terraform_remote_state.infrastructure.outputs.cluster_location
}

resource "kubernetes_namespace" "namespace" {
  count = data.terraform_remote_state.infrastructure.outputs.kubernetes_service_account_namespace == "default" ? 0 : 1

  metadata {
    name = data.terraform_remote_state.infrastructure.outputs.kubernetes_service_account_namespace
  }
}

resource "kubernetes_service_account" "service_account" {
  depends_on = [kubernetes_namespace.namespace]
  count = data.terraform_remote_state.infrastructure.outputs.kubernetes_service_account_name == "default" ? 0 : 1

  metadata {
    name = data.terraform_remote_state.infrastructure.outputs.kubernetes_service_account_name
    namespace = data.terraform_remote_state.infrastructure.outputs.kubernetes_service_account_namespace
  }
}

resource "kubernetes_manifest" "image_pull_secret" {
  provider = kubernetes-alpha
  depends_on = [kubernetes_namespace.namespace]

  manifest = {
    apiVersion = "example.apstn.dev/v1alpha1"
    kind       = "ImagePullSecret"
    metadata = {
      name      = "imagepullsecret-sample"
      namespace = data.terraform_remote_state.infrastructure.outputs.kubernetes_service_account_namespace
    }
    spec = {
      secretName : "image-pull-secret"
      serviceAccountName = data.terraform_remote_state.infrastructure.outputs.kubernetes_service_account_name
      workloadIdentityPoolProvider : data.terraform_remote_state.infrastructure.outputs.workload_identity_pool_provider_name
      gsaEmail : data.terraform_remote_state.infrastructure.outputs.image_puller_email
    }
  }
}