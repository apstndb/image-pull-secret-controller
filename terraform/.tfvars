# Must set billing account ID
# billing_account_id = "012345-6789AB-CDEF01"

# set prefix at most 25 chars long
cluster_project_prefix = "yourname-example-cluster"
service_project_prefix = "yourname-example-service"

cluster_name     = "image-pull-test"
cluster_location = "us-central1"
kubernetes_service_account_name = "default"
kubernetes_service_account_namespace = "default"
image_puller_gsa_name = "image-puller"
enable_autopilot = true

workload_identity_pool_id = "pool-for-gke"
workload_identity_pool_provider_id = "provider-for-gke"
artifact_registry_repo_name = "repo"
