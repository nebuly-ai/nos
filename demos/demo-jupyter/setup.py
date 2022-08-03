from setuptools import setup

setup(
    name="jupyter-n8s",
    version="0.0.1",
    entry_points={
        'jupyter_client.kernel_provisioners': [
            'k8s-provisioner=n8s.provisioner:K8sKernelProvisioner',
        ],
    },
    install_requires=[
        "jupyter_server==1.18.1",
    ]
)
