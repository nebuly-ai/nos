from typing import List, Any

from jupyter_client import LocalProvisioner, KernelConnectionInfo


class K8sKernelProvisioner(LocalProvisioner):

    async def launch_kernel(self, cmd: List[str], **kwargs: Any) -> KernelConnectionInfo:
        print(cmd)
        return await super(K8sKernelProvisioner, self).launch_kernel(cmd, **kwargs)
