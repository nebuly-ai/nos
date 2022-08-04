import logging

from jupyter_server.gateway.gateway_client import GatewayClient
from jupyter_server.gateway.managers import GatewayKernelManager

from kernel import KernelClient


def test_remote_code_execution():
    import asyncio

    gateway_client = GatewayClient.instance()
    gateway_client.url = "http://nebulyaks.westeurope.cloudapp.azure.com"
    kernel_manager = GatewayKernelManager()

    coro = kernel_manager.start_kernel(kernel_name="python_kubernetes")
    asyncio.run(coro)

    kernel_client = KernelClient(
        http_api_endpoint="http://nebulyaks.westeurope.cloudapp.azure.com/api/kernels",
        ws_api_endpoint="ws://nebulyaks.westeurope.cloudapp.azure.com/api/kernels",
        kernel_id=kernel_manager.kernel_id,
        logger=logging.getLogger(__name__)
    )

    resp = kernel_client.execute("""
    from time import sleep
    for i in range(10):
        print("hello world")
        sleep(1)
    a = 5
    """)
    print(resp)
