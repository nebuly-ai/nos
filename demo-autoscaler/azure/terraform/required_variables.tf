### General ###
variable "tenant_id" {
  type        = string
  description = "Azure Tenant ID."
}
variable "subscription_id" {
  type        = string
  description = "Azure Subscription ID."
}
variable "client_id" {
  type        = string
  description = "Application ID of the Service Principal used for provisioning resources."
}
variable "client_secret" {
  type        = string
  description = "Client secret of the Service Principal used for provisioning resources."
}
variable "resource_group_name" {
  type = string
  description = "The name of the resource group in which resources will be provisioned."
}
