# Troubleshooting

You can check the logs of the scheduler by running the following command:

```shell
 kubectl logs -n nebuly-nos -l app.kubernetes.io/component=nos-scheduler -f
```

You can check the logs of the operator by running the following command:

```shell
 kubectl logs -n nebuly-nos -l app.kubernetes.io/component=nos-operator -f
```
