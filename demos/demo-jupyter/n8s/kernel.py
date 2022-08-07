import contextlib
import dataclasses
import enum
import logging
import os
import ssl
import traceback
from typing import Dict, Union, List, Optional
from uuid import uuid4

import requests
import websockets
from ipykernel.ipkernel import IPythonKernel
from jupyter_server.gateway.gateway_client import GatewayClient
from jupyter_server.gateway.managers import GatewayKernelManager
from tornado.escape import json_encode, json_decode, utf8

log_level = os.getenv("LOG_LEVEL", "DEBUG")
REQUEST_TIMEOUT = int(os.getenv("REQUEST_TIMEOUT", 10))


class MessageType(str, enum.Enum):
    ERROR = "error"
    STREAM = "stream"
    EXECUTE_RESULT = "execute_result"
    DISPLAY_DATA = "display_data"
    STATUS = "status"
    EXECUTE_INPUT = "execute_input"
    EXECUTE_REPLY = "execute_reply"


class ExecuteStatus(str, enum.Enum):
    OK = "ok"
    ERROR = "error"


@dataclasses.dataclass
class ExecuteResult:
    status: ExecuteStatus
    exception_name: str = None
    exception_value: str = None
    exception_traceback: List[str] = None

    def is_ok(self) -> bool:
        return self.status is ExecuteStatus.OK


def _message_contains_error(msg_type: MessageType, msg: dict) -> bool:
    if msg_type == MessageType.ERROR:
        return True
    if msg_type == MessageType.EXECUTE_RESULT and msg['content']['status'] == 'error':
        return True
    return False


def _extract_message_id(msg: dict) -> Optional[str]:
    if "msg_id" in msg["parent_header"]:
        return msg["parent_header"]["msg_id"]
    # msg_id may not be in the parent_header, see if present in response
    # IPython kernel appears to do this after restarts with a 'starting' status
    return msg.get("msg_id", None)


@dataclasses.dataclass
class ExecuteRespMessage:
    message_type: MessageType
    message_id: str
    exception_name: str = None
    exception_value: str = None
    exception_traceback: List[str] = None
    execution_state: str = None
    content: str = None

    def is_idle(self):
        return self.execution_state == "idle"

    def is_error(self):
        return self.exception_name is not None

    def is_stream(self):
        return self.message_type is MessageType.STREAM

    def is_response_of(self, msg_id: str) -> bool:
        """Returns True if the message is a response to the message with ID provided as argument.
        """
        return self.message_id == msg_id

    def __repr__(self) -> str:
        return '[id: "{}" type: "{}" execution_state: "{}"]'.format(
            self.message_id,
            self.message_type,
            self.execution_state,
        )

    @staticmethod
    def from_raw_message(raw_msg: str) -> "ExecuteRespMessage":
        msg = json_decode(utf8(raw_msg))
        msg_type = MessageType(msg['msg_type'])
        msg_id = _extract_message_id(msg)
        resp = ExecuteRespMessage(msg_type, msg_id)

        if _message_contains_error(msg_type, msg) is True:
            resp.exception_name = msg["content"]["ename"]
            resp.exception_value = msg["content"]["evalue"]
            resp.exception_traceback = msg["content"]["traceback"]
            return resp

        if msg_type is MessageType.STREAM:
            resp.content = msg['content']['text']
            return resp

        if msg_type in [MessageType.EXECUTE_RESULT, MessageType.DISPLAY_DATA]:
            if 'text/plain' in msg['content']['data']:
                resp.content = msg['content']['data']['text/plain']
            elif 'text/html' in msg['content']['data']:
                resp.content = msg['content']['data']['text/html']
            return resp

        if msg_type is MessageType.STATUS:
            resp.execution_state = msg['content']['execution_state']
            return resp

        return resp


class KernelClient(object):
    DEAD_MSG_ID = 'deadbeefdeadbeefdeadbeefdeadbeef'
    POST_IDLE_TIMEOUT = 0.5
    DEFAULT_INTERRUPT_WAIT = 1

    def __init__(
            self,
            http_api_endpoint: str,
            ws_api_endpoint: str,
            kernel_id: str,
            logger=None
    ):
        self.shutting_down = False
        self.restarting = False
        self.http_api_endpoint = http_api_endpoint
        self.kernel_http_api_endpoint = '{}/{}'.format(http_api_endpoint, kernel_id)
        self.ws_api_endpoint = ws_api_endpoint
        self.kernel_ws_api_endpoint = '{}/{}/channels'.format(ws_api_endpoint, kernel_id)
        self.kernel_id = kernel_id
        self.logger = logger

    async def execute(self, code: Union[str, List[str]]):
        """
        Executes the code provided and returns the result of that execution.
        """
        self.logger.debug('Sending execute request to kernel {} to {}'.format(
            self.kernel_id,
            self.kernel_ws_api_endpoint)
        )

        ctx = ssl.create_default_context()
        ctx.check_hostname = False
        ctx.verify_mode = ssl.VerifyMode.CERT_NONE

        async with websockets.connect(uri=self.kernel_ws_api_endpoint, ssl=ctx) as connection:
            # Send execute request
            msg_id = uuid4().hex
            req = self.__new_execute_request(msg_id, code)
            await connection.send(req)
            # Listen for responses
            async for raw_msg in connection:
                msg = ExecuteRespMessage.from_raw_message(raw_msg)
                if msg.is_response_of(msg_id):
                    self.logger.debug("Received response message {}".format(msg))
                    if msg.is_idle() or msg.is_error():
                        # If the Kernel is idle or there's an error, then stop listening for response messages
                        yield msg
                        break
                    yield msg
                else:
                    self.logger.debug("Received message {}, ignoring it".format(msg))

    def interrupt(self):
        url = "{}/{}".format(self.kernel_http_api_endpoint, "interrupt")
        response = requests.post(url)
        if response.status_code == 204:
            self.logger.debug('Kernel {} interrupted'.format(self.kernel_id))
            return True
        else:
            raise RuntimeError('Unexpected response interrupting kernel {}: {}'.
                               format(self.kernel_id, response.content))

    def restart(self, timeout=REQUEST_TIMEOUT):
        self.restarting = True
        self.kernel_socket.close()
        self.kernel_socket = None
        url = "{}/{}".format(self.kernel_http_api_endpoint, "restart")
        response = requests.post(url)
        if response.status_code == 200:
            self.logger.debug('Kernel {} restarted'.format(self.kernel_id))
            self.kernel_socket = \
                websocket.create_connection(self.kernel_ws_api_endpoint, timeout=timeout, enable_multithread=True)
            self.restarting = False
            return True
        else:
            self.restarting = False
            raise RuntimeError('Unexpected response restarting kernel {}: {}'.format(self.kernel_id, response.content))

    def get_state(self):
        url = "{}".format(self.kernel_http_api_endpoint)
        response = requests.get(url)
        if response.status_code == 200:
            json = response.json()
            self.logger.debug('Kernel {} state: {}'.format(self.kernel_id, json))
            return json['execution_state']
        else:
            raise RuntimeError('Unexpected response retrieving state for kernel {}: {}'.
                               format(self.kernel_id, response.content))

    @staticmethod
    def __new_execute_request(msg_id: str, code: Union[str, List[str]]) -> str:
        return json_encode({
            'header': {
                'username': '',
                'version': '5.0',
                'session': '',
                'msg_id': msg_id,
                'msg_type': 'execute_request'
            },
            'parent_header': {},
            'channel': 'shell',
            'content': {
                'code': "".join(code),
                'silent': False,
                'store_history': False,
                'user_expressions': {},
                'allow_stdin': False
            },
            'metadata': {},
            'buffers': {}
        })


class NebulnetesKernel(IPythonKernel):
    banner = "Nebulnetes kernel 🚀"
    GATEWAY_HOST = os.getenv("GATEWAY_BASE_ADDRESS", "127.0.0.1")
    N8S_MAGIC_COMMAND = "%%n8s"

    def __init__(self, **kwargs):
        super().__init__(**kwargs)
        self.logger = logging.getLogger(__name__)
        self.logger.setLevel(log_level)
        logging.basicConfig()
        self.kernel_clients: Dict[str, KernelClient] = {}
        self.__gateway_http_url = "https://{}".format(self.GATEWAY_HOST)
        self.__gateway_ws_url = "wss://{}".format(self.GATEWAY_HOST)
        self._init_gateway_client()
        self.logger.debug("Jupyter Enterprise Gateway endpoints:\nhttp: {}\nws: {}".format(
            self.__gateway_http_url,
            self.__gateway_ws_url,
        ))

    def _init_gateway_client(self):
        gateway_client = GatewayClient.instance()
        gateway_client.url = self.__gateway_http_url
        gateway_client.request_timeout = 120
        # Disable SSL certs verification, only for dev purposes
        gateway_client.init_static_args()
        gateway_client._static_args["validate_cert"] = False  # noqa

    def _new_kernel_client(self, kernel_id: str) -> KernelClient:
        return KernelClient(
            http_api_endpoint="https://{}/api/kernels".format(self.GATEWAY_HOST),
            ws_api_endpoint="wss://{}/api/kernels".format(self.GATEWAY_HOST),
            kernel_id=kernel_id,
            logger=self.logger
        )

    async def _start_new_remote_kernel(self) -> GatewayKernelManager:
        kernel_manager = GatewayKernelManager()
        kernel_manager.logger = self.logger
        self.logger.info(f"Starting new kernel on {self.GATEWAY_HOST}...")
        await kernel_manager.start_kernel(kernel_name="python_kubernetes")
        return kernel_manager

    @contextlib.asynccontextmanager
    async def _new_remote_kernel(self):
        kernel_manager = await self._start_new_remote_kernel()
        try:
            yield self._new_kernel_client(kernel_manager.kernel_id)
        finally:
            self.logger.debug("Shutting down kernel {}...".format(kernel_manager.kernel_id))
            await kernel_manager.shutdown_kernel(now=True)
            self.logger.debug("Shut down kernel {}".format(kernel_manager.kernel_id))

    async def _handle_execute_resp(self, msg: ExecuteRespMessage):
        if msg.is_stream():
            content = {"name": "stdout", "text": msg.content}
            self.send_response(self.iopub_socket, MessageType.STREAM.value, content)
        if msg.is_error():
            content = {
                "ename": msg.exception_name,
                "evalue": msg.exception_name,
                "traceback": msg.exception_traceback,
            }
            self.send_response(self.iopub_socket, MessageType.ERROR.value, content)

    async def _do_execute_on_remote_kernel(self, code):
        async with self._new_remote_kernel() as kernel_client:
            async for msg in kernel_client.execute(code):
                await self._handle_execute_resp(msg)
                if msg.is_error():
                    return {
                        "status": ExecuteStatus.ERROR.value,
                        "ename": msg.exception_name,
                        "evalue": msg.exception_name,
                        "traceback": msg.exception_traceback,
                    }
            return {
                "status": ExecuteStatus.OK.value,
                "execution_count": self.execution_count,
                "user_expressions": {},
                "payload": []
            }

    async def do_execute(self, code, silent, store_history=True, user_expressions=None, allow_stdin=False, **kwargs):
        code_lines = code.split("\n")
        if code_lines[0] != self.N8S_MAGIC_COMMAND:
            return await super().do_execute(code, silent, store_history, user_expressions, allow_stdin, **kwargs)

        try:
            self.execution_count += 1
            code = "\n".join(code_lines[1:])  # remove magic command from code
            return await self._do_execute_on_remote_kernel(code)
        except Exception as e:
            self.logger.exception(e)
            return {
                "status": "error",
                "ename": type(e).__name__,
                "execution_count": self.execution_count,
                "evalue": str(e),
                "traceback": traceback.format_tb(e.__traceback__)
            }

    def do_shutdown(self, restart):
        return super().do_shutdown(restart)
