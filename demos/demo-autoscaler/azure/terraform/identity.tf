resource "azuread_application" "cluster_autoscaler" {
  display_name = "${var.resource_prefix}.cluster-autoscaler"
  owners       = [data.azuread_client_config.current.object_id]
}

resource "azuread_service_principal" "cluster_autoscaler" {
  application_id               = azuread_application.cluster_autoscaler.application_id
  app_role_assignment_required = true
  owners                       = [data.azuread_client_config.current.object_id]
}

resource "azuread_service_principal_password" "cluster_autoscaler" {
  service_principal_id = azuread_service_principal.cluster_autoscaler.object_id
}
