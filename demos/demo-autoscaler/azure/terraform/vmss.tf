resource "tls_private_key" "main" {
  algorithm = "RSA"
  rsa_bits  = 4096
}

resource "azurerm_linux_virtual_machine_scale_set" "main" {
  name                = "${var.resource_prefix}vmss"
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location
  sku                 = "Standard_D2ads_v5"
  instances           = 0

  admin_username = var.vmss_admin_username
  admin_ssh_key {
    public_key = tls_private_key.main.public_key_openssh
    username   = var.vmss_admin_username
  }

  source_image_reference {
    publisher = "Canonical"
    offer     = "UbuntuServer"
    sku       = "18.04-LTS"
    version   = "latest"
  }

  os_disk {
    storage_account_type = "Standard_LRS"
    caching              = "ReadWrite"
  }

  network_interface {
    name    = "${var.resource_prefix}ni"
    primary = true

    ip_configuration {
      name      = "primary-ip-config"
      primary   = true
      subnet_id = azurerm_subnet.main.id

      public_ip_address {
        name = "${var.resource_prefix}ip"
      }
    }
  }

  tags = {
    ### Tags for cluster-autoscaler nodes labels ###
    "ki8s.io_cluster-autoscaler_node-template_label_foo" = "bar"
    ### Tags for cluster-autoscaler nodes taints ###
    "autoscaler_node-template_taint_foo" = "unbar:NoSchedule"
    ### Tags for cluster-autoscaler auto-discovery ###
    "cluster-autoscaler-enabled" = true
    "cluster-autoscaler-name"    = "kind-anton"
    ### Tags for cluster-autoscaler min/max nodes"
    "min"      = 1
    "max"      = 10
    "poolName" = "${var.resource_prefix}vmss"
  }
}
