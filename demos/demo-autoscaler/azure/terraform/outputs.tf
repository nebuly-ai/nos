output "cluster_autoscaler_sp" {
  value = {
    client_id     = azuread_application.cluster_autoscaler.application_id
    client_secret = azuread_service_principal_password.cluster_autoscaler.value
  }
  sensitive = true
}
