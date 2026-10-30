# MySQL through an SSH port forward

This example starts `ssh -N -L` on the machine running Terraform or OpenTofu. The MySQL provider connects to the forwarded local port and creates a user. The tunnel remains open while the dependent resource is processed and closes when the operation ends.

Set `ssh_host`, `ssh_user`, `mysql_host`, `mysql_admin_user`, `mysql_admin_password`, and `app_password`. SSH must already be able to authenticate to the bastion without prompting. `mysql_host` is resolved from the bastion. Change `local_port` if port 13306 is occupied.

The process passes the local endpoint through `inputs` to its computed `outputs`. The MySQL provider references that output, so its configuration depends on the tunnel opening and completing its wait. `wait_for_startup_delay_ms` gives SSH one second to report a bind failure; it does not check that MySQL is ready. The MySQL provider retries connections. Creating a user requires suitable privileges, and the user password must be protected in Terraform state.

Once the subprocess provider is listed in the Terraform Registry, run from [`examples/mysql-port-forward`](https://github.com/alexanderankin/terraform-provider-subprocess/tree/main/examples/mysql-port-forward):

```sh
tofu init
tofu apply \
  -var='bastion_host=bastion.example.com' \
  -var='ssh_user=operator' \
  -var='mysql_host=mysql.internal' \
  -var='mysql_admin_user=admin'
```

Supply the two sensitive passwords through `TF_VAR_mysql_admin_password` and `TF_VAR_app_password` in your environment.

```hcl
--8<-- "examples/mysql-port-forward/main.tf"
```
