import logging

import websocket
from jupyter_server.gateway.gateway_client import GatewayClient
from jupyter_server.gateway.managers import GatewayKernelManager

from .kernel import KernelClient


def test_remote_code_execution():
    import asyncio

    gateway_client = GatewayClient.instance()
    gateway_client.init_static_args()
    gateway_client._static_args["validate_cert"] = False
    gateway_client.url = "https://nebulyaks.westeurope.cloudapp.azure.com"
    kernel_manager = GatewayKernelManager()

    # coro = kernel_manager.start_kernel(kernel_name="python_kubernetes")
    # asyncio.run(coro)

    kernel_client = KernelClient(
        http_api_endpoint="https://nebulyaks.westeurope.cloudapp.azure.com/api/kernels",
        ws_api_endpoint="wss://nebulyaks.westeurope.cloudapp.azure.com/api/kernels",
        kernel_id="4a48d6be-0e95-414a-ad41-c05a8e36329e",
        logger=logging.getLogger(__name__)
    )

    asyncio.run(kernel_client.execute("""
    asd)
    from time import sleep
    for i in range(10):
        print("hello world")
        sleep(1)
    """))


def test_websocket_connection():
    try:
        socket = websocket.create_connection(
            "wss://nebulyaks.westeurope.cloudapp.azure.com/api/kernels/8f71478b-fef4-4a40-ada2-a19eab0a0d72/channels",
            timeout=10,
            enable_multithread=True,
        )
        socket.close()
    except Exception as e:
        pass
