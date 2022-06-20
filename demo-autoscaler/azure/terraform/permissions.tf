resource "azurerm_role_assignment" "cluster_autoscaler_vmss_contributor" {
  scope                = azurerm_linux_virtual_machine_scale_set.main.id
  role_definition_name = "Contributor"
  principal_id         = azuread_service_principal.cluster_autoscaler.object_id
}
