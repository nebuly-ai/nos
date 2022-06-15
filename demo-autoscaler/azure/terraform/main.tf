### Init providers ###
terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~>3.0.0"
    }
    azuread = {
      source  = "hashicorp/azuread"
      version = ">=2.0.0"
    }
    tls = {

    }
  }
}

provider "azurerm" {
  features {}
}

### Resource group ###
resource "azurerm_resource_group" "main" {
  name     = var.resource_group_name
  location = var.location
}
