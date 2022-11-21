# How to pull Nebulnetes images

Since at the moment Nebulnetes is not publicly available, you need to use provide credentials for pulling the Docker
images from our private GitHub container registry.

The k8s manifests for deploying Nebulnetes components expect the Docker credentials to be stored in a secret
called `ghcr` in the namespace `n8s-system`. You can create the secret by following these steps:

1. Create a personal access token with the `read:packages` scope on GitHub. You can find more information on how to
   create a personal access
   token [here](https://docs.github.com/en/github/authenticating-to-github/creating-a-personal-access-token).
2. Create an auth string with the following format: `<username>:<token>`, where `<username>` is your GitHub username
   and `<token>` is the personal access token you created in the previous step.
3. Encode the auth string in base64: `echo -n "<auth_string>" | base64`
4. Use the base64-encoded auth string of the previous step to create a `.dockerconfig.json` file with the following
   format:
   ```json
   {
     "auths": {
       "ghcr.io": {
         "auth": "<base64 auth string>"
       }
     }
   }
   ```
5. Encode the `.dockerconfig.json` file in base64: `cat .dockerconfig.json | base64`
6. Create a `ghcr-secret.yaml` file with the k8s secret manifest containing the encoded credential of the previous step
   ```yaml
    kind: Secret
    type: kubernetes.io/dockerconfigjson
    apiVersion: v1
    metadata:
       name: ghcr
       namespace: n8s-system
    data:
       .dockerconfigjson: <previous-step-output>
    ```
7. Create the k8s secret
   ```shell
    kubectl apply -f ghr-secret.yaml
   ```