variable "cluster_name" {}
variable "cluster_location" {}
variable "workload_identity_pool_id" {}
variable "workload_identity_pool_provider_id" {}
variable "cluster_project_prefix" {}
variable "service_project_prefix" {}
variable "kubernetes_service_account_name" {}
variable "kubernetes_service_account_namespace" {}
variable "image_puller_gsa_name" {}
variable "artifact_registry_repo_name" {}
variable "org_id" {
  type    = string
  default = ""
}
variable "folder_id" {
  type    = string
  default = ""
}
variable "billing_account_id" {}
variable "enable_autopilot" {
  type = bool
}

provider "google" {
}

provider "google-beta" {
}

module "cluster_project" {
  source  = "terraform-google-modules/project-factory/google"
  version = "~> 10.1"

  billing_account         = var.billing_account_id
  org_id                  = var.org_id
  folder_id               = var.folder_id
  name                    = var.cluster_project_prefix
  random_project_id       = true
  default_service_account = "keep"
  auto_create_network     = true
  activate_apis = [
    "container.googleapis.com",
  ]
}


module "service_project" {
  source  = "terraform-google-modules/project-factory/google"
  version = "~> 10.1"

  billing_account   = var.billing_account_id
  org_id            = var.org_id
  folder_id         = var.folder_id
  name              = var.service_project_prefix
  random_project_id = true
  activate_apis = [
    "artifactregistry.googleapis.com",
    "iamcredentials.googleapis.com",
  ]
}

resource "google_service_account" "image_puller" {
  provider   = google
  project    = module.service_project.project_id
  account_id = var.image_puller_gsa_name
}

locals {
  kubernetes_service_account_subject   = "system:serviceaccount:${var.kubernetes_service_account_namespace}:${var.kubernetes_service_account_name}"
  workload_identity_pool_name          = "projects/${module.service_project.project_number}/locations/global/workloadIdentityPools/${google_iam_workload_identity_pool.gke_pool.workload_identity_pool_id}"
  workload_identity_pool_provider_name = "${local.workload_identity_pool_name}/providers/${google_iam_workload_identity_pool_provider.gke_pool.workload_identity_pool_provider_id}"
}
resource "google_service_account_iam_member" "image_puller_binding" {
  provider           = google
  service_account_id = google_service_account.image_puller.id
  member             = "principal://iam.googleapis.com/${local.workload_identity_pool_name}/subject/${local.kubernetes_service_account_subject}"
  role               = "roles/iam.workloadIdentityUser"
}

resource "google_artifact_registry_repository_iam_member" "image_puller_is_artifact_registry_reader" {
  provider   = google-beta
  role       = "roles/artifactregistry.reader"
  member     = "serviceAccount:${google_service_account.image_puller.email}"
  project    = google_artifact_registry_repository.repo.project
  location   = google_artifact_registry_repository.repo.location
  repository = google_artifact_registry_repository.repo.name
}

resource "google_iam_workload_identity_pool" "gke_pool" {
  provider                  = google-beta
  project                   = module.service_project.project_id
  workload_identity_pool_id = var.workload_identity_pool_id
}

resource "google_iam_workload_identity_pool_provider" "gke_pool" {
  provider                           = google-beta
  project                            = module.service_project.project_id
  workload_identity_pool_id          = google_iam_workload_identity_pool.gke_pool.workload_identity_pool_id
  workload_identity_pool_provider_id = var.workload_identity_pool_provider_id
  attribute_mapping = {
    "google.subject" = "assertion.sub"
  }
  oidc {
    allowed_audiences = []
    issuer_uri        = "https://container.googleapis.com/v1/projects/${google_container_cluster.cluster.project}/locations/${google_container_cluster.cluster.location}/clusters/${google_container_cluster.cluster.name}"
  }
}

resource "google_artifact_registry_repository" "repo" {
  provider = google-beta
  project  = module.service_project.project_id

  labels        = {}
  location      = "us-central1"
  repository_id = var.artifact_registry_repo_name
  description   = "private docker repository"
  format        = "DOCKER"
}

resource "google_container_cluster" "cluster" {
  provider         = google
  project          = module.cluster_project.project_id
  name             = var.cluster_name
  location         = var.cluster_location
  enable_autopilot = var.enable_autopilot
  vertical_pod_autoscaling {
    enabled = true
  }
  initial_node_count = var.enable_autopilot ? null : 1
}


output "image_puller_email" {
  value = google_service_account.image_puller.email
}

output "repo" {
  value = "${google_artifact_registry_repository.repo.location}-docker.pkg.dev/${google_artifact_registry_repository.repo.project}/${google_artifact_registry_repository.repo.repository_id}"
}

output "cluster_project_id" {
  value = module.cluster_project.project_id
}

output "service_project_id" {
  value = module.service_project.project_id
}

output "workload_identity_pool_provider_name" {
  value = local.workload_identity_pool_provider_name
}

output "cluster_name" {
  value = google_container_cluster.cluster.name
}

output "cluster_location" {
  value = google_container_cluster.cluster.location
}

output "kubernetes_service_account_name" {
  value = var.kubernetes_service_account_name
}

output "kubernetes_service_account_namespace" {
  value = var.kubernetes_service_account_namespace
}

output "get_credentials_cmd" {
  value = "gcloud container clusters get-credentials --project ${google_container_cluster.cluster.project} ${google_container_cluster.cluster.name} ${
    length(regexall(".*-[a-z]$", google_container_cluster.cluster.location)) > 0
    ? "--zone=${google_container_cluster.cluster.location}"
    : "--region=${google_container_cluster.cluster.location}"
  }"
}
