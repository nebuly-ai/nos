import logging
import os
import queue
import time
from threading import Thread
from uuid import uuid4

import requests
import websocket
from ipykernel.ipkernel import IPythonKernel
from jupyter_server.gateway.gateway_client import GatewayClient
from jupyter_server.gateway.managers import GatewayKernelManager
from tornado.escape import json_encode, json_decode, utf8

log_level = os.getenv("LOG_LEVEL", "DEBUG")
REQUEST_TIMEOUT = int(os.getenv("REQUEST_TIMEOUT", 10))


class KernelClient(object):
    DEAD_MSG_ID = 'deadbeefdeadbeefdeadbeefdeadbeef'
    POST_IDLE_TIMEOUT = 0.5
    DEFAULT_INTERRUPT_WAIT = 1

    def __init__(self, http_api_endpoint, ws_api_endpoint, kernel_id, timeout=REQUEST_TIMEOUT, logger=None):
        self.shutting_down = False
        self.restarting = False
        self.http_api_endpoint = http_api_endpoint
        self.kernel_http_api_endpoint = '{}/{}'.format(http_api_endpoint, kernel_id)
        self.ws_api_endpoint = ws_api_endpoint
        self.kernel_ws_api_endpoint = '{}/{}/channels'.format(ws_api_endpoint, kernel_id)
        self.kernel_id = kernel_id
        self.log = logger
        self.log.debug('Initializing kernel client ({}) to {}'.format(kernel_id, self.kernel_ws_api_endpoint))

        self.kernel_socket = websocket.create_connection(self.kernel_ws_api_endpoint, timeout=timeout,
                                                         enable_multithread=True)

        self.response_queues = {}

        # startup reader thread
        self.response_reader = Thread(target=self._read_responses)
        self.response_reader.start()
        self.interrupt_thread = None

    def shutdown(self):
        # Terminate thread, close socket and clear queues.
        self.shutting_down = True

        if self.kernel_socket:
            self.kernel_socket.close()
            self.kernel_socket = None

        if self.response_queues:
            self.response_queues.clear()
            self.response_queues = None

        if self.response_reader:
            self.response_reader.join(timeout=2.0)
            if self.response_reader.is_alive():
                self.log.warning("Response reader thread is not terminated, continuing...")
            self.response_reader = None

    def execute(self, code, timeout=REQUEST_TIMEOUT):
        """
        Executes the code provided and returns the result of that execution.
        """
        response = []
        try:
            msg_id = self._send_request(code)

            post_idle = False
            while True:
                response_message = self._get_response(msg_id, timeout, post_idle)
                if response_message:
                    response_message_type = response_message['msg_type']

                    if response_message_type == 'error' or \
                            (response_message_type == 'execute_reply' and
                             response_message['content']['status'] == 'error'):
                        response.append('{}:{}:{}'.format(response_message['content']['ename'],
                                                          response_message['content']['evalue'],
                                                          response_message['content']['traceback']))
                    elif response_message_type == 'stream':
                        response.append(KernelClient._convert_raw_response(response_message['content']['text']))

                    elif response_message_type == 'execute_result' or response_message_type == 'display_data':
                        if 'text/plain' in response_message['content']['data']:
                            response.append(
                                KernelClient._convert_raw_response(response_message['content']['data']['text/plain']))
                        elif 'text/html' in response_message['content']['data']:
                            response.append(
                                KernelClient._convert_raw_response(response_message['content']['data']['text/html']))
                    elif response_message_type == 'status':
                        if response_message['content']['execution_state'] == 'idle':
                            post_idle = True  # indicate we're at the logical end and timeout poll for next message
                            continue
                    else:
                        self.log.debug("Unhandled response for msg_id: {} of msg_type: {}".
                                       format(msg_id, response_message_type))

                if response_message is None:  # We timed out.  If post idle, its ok, else make mention of it
                    if not post_idle:
                        self.log.warning("Unexpected timeout occurred for msg_id: {} - no 'idle' status received!".
                                         format(msg_id))
                    break

        except BaseException as b:
            self.log.debug(b)

        return ''.join(response)

    def interrupt(self):
        url = "{}/{}".format(self.kernel_http_api_endpoint, "interrupt")
        response = requests.post(url)
        if response.status_code == 204:
            self.log.debug('Kernel {} interrupted'.format(self.kernel_id))
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
            self.log.debug('Kernel {} restarted'.format(self.kernel_id))
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
            self.log.debug('Kernel {} state: {}'.format(self.kernel_id, json))
            return json['execution_state']
        else:
            raise RuntimeError('Unexpected response retrieving state for kernel {}: {}'.
                               format(self.kernel_id, response.content))

    def start_interrupt_thread(self, wait_time=DEFAULT_INTERRUPT_WAIT):
        self.interrupt_thread = Thread(target=self.perform_interrupt, args=(wait_time,))
        self.interrupt_thread.start()

    def perform_interrupt(self, wait_time):
        time.sleep(wait_time)  # Allow parent to start executing cell to interrupt
        self.interrupt()

    def terminate_interrupt_thread(self):
        if self.interrupt_thread:
            self.interrupt_thread.join()
            self.interrupt_thread = None

    def _send_request(self, code):
        """
        Builds the request and submits it to the kernel.  Prior to sending the request it
        creates an empty response queue and adds it to the dictionary using msg_id as the
        key.  The msg_id is returned in order to read responses.
        """
        msg_id = uuid4().hex
        message = KernelClient.__create_execute_request(msg_id, code)

        # create response-queue and add to map for this msg_id
        self.response_queues[msg_id] = queue.Queue()

        self.kernel_socket.send(message)

        return msg_id

    def _get_response(self, msg_id, timeout, post_idle):
        """
        Pulls the next response message from the queue corresponding to msg_id.  If post_idle is true,
        the timeout parameter is set to a very short value since a majority of time, there won't be a
        message in the queue.  However, in cases where a race condition occurs between the idle status
        and the execute_result payload - where the two are out of order, then this will pickup the result.
        """

        if post_idle and timeout > KernelClient.POST_IDLE_TIMEOUT:
            timeout = KernelClient.POST_IDLE_TIMEOUT  # overwrite timeout to small value following idle messages.

        msg_queue = self.response_queues.get(msg_id)
        try:
            self.log.debug("Getting response for msg_id: {} with timeout: {}".format(msg_id, timeout))
            response = msg_queue.get(timeout=timeout)
            self.log.debug("Got response for msg_id: {}, msg_type: {}".
                           format(msg_id, response['msg_type'] if response else 'null'))
        except queue.Empty:
            response = None

        return response

    def _read_responses(self):
        """
        Reads responses from the websocket.  For each response read, it is added to the response queue based
        on the messages parent_header.msg_id.  It does this for the duration of the class's lifetime until its
        shutdown method is called, at which time the socket is closed (unblocking the reader) and the thread
        terminates.  If shutdown happens to occur while processing a response (unlikely), termination takes
        place via the loop control boolean.
        """
        try:
            while not self.shutting_down:
                try:
                    raw_message = self.kernel_socket.recv()
                    response_message = json_decode(utf8(raw_message))

                    msg_id = KernelClient._get_msg_id(response_message, self.log)

                    if msg_id not in self.response_queues:
                        # this will happen when the msg_id is generated by the server
                        self.response_queues[msg_id] = queue.Queue()

                    # insert into queue
                    self.log.debug("Inserting response for msg_id: {}, msg_type: {}".
                                   format(msg_id, response_message['msg_type']))
                    self.response_queues.get(msg_id).put_nowait(response_message)
                except BaseException as be1:
                    if self.restarting:  # If restarting, wait until restart has completed - which includes new socket
                        i = 1
                        while self.restarting:
                            if i >= 10 and i % 2 == 0:
                                self.log.debug("Still restarting after {} secs...".format(i))
                            time.sleep(1)
                            i += 1
                        continue
                    raise be1

        except websocket.WebSocketConnectionClosedException:
            pass  # websocket closure most likely due to shutdown

        except BaseException as be2:
            if not self.shutting_down:
                self.log.warning('Unexpected exception encountered ({})'.format(be2))

        self.log.debug('Response reader thread exiting...')

    @staticmethod
    def _get_msg_id(message, logger):
        msg_id = KernelClient.DEAD_MSG_ID
        if message:
            if 'msg_id' in message['parent_header'] and message['parent_header']['msg_id']:
                msg_id = message['parent_header']['msg_id']
            elif 'msg_id' in message:
                # msg_id may not be in the parent_header, see if present in response
                # IPython kernel appears to do this after restarts with a 'starting' status
                msg_id = message['msg_id']
        else:  # Dump the "dead" message...
            logger.debug("+++++ Dumping dead message: {}".format(message))
        return msg_id

    @staticmethod
    def _convert_raw_response(raw_response_message):
        result = raw_response_message
        if isinstance(raw_response_message, str):
            if "u'" in raw_response_message:
                result = raw_response_message.replace("u'", "")[:-1]

        return result

    @staticmethod
    def __create_execute_request(msg_id, code):
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
    # implementation = 'Echo'
    # implementation_version = '1.0'
    # language = 'no-op'
    # language_version = '0.1'
    # language_info = {
    #     'name': 'Any text',
    #     'mimetype': 'text/plain',
    #     'file_extension': '.txt',
    # }
    banner = "Nebulnetes kernel 🚀"

    def __init__(self, **kwargs):
        super().__init__(**kwargs)
        self.logger = logging.getLogger(__name__)
        self.logger.setLevel(log_level)
        logging.basicConfig()
        gateway_client = GatewayClient.instance()
        gateway_client.url = "http://nebulyaks.westeurope.cloudapp.azure.com"

    async def do_execute(self, code, silent, store_history=True, user_expressions=None, allow_stdin=False, **kwargs):
        reply_content = {
            'status': 'ok',
            # The base class increments the execution count
            'execution_count': 1,
            'payload': [],
            'user_expressions': {},
        }
        kernel_manager = GatewayKernelManager()
        kernel_manager.log = self.logger

        await kernel_manager.start_kernel(kernel_name="python_kubernetes")
        kernel_client = KernelClient(
            http_api_endpoint="http://nebulyaks.westeurope.cloudapp.azure.com/api/kernels",
            ws_api_endpoint="ws://nebulyaks.westeurope.cloudapp.azure.com/api/kernels",
            kernel_id=kernel_manager.kernel_id,
            logger=self.logger
        )
        resp = kernel_client.execute(code, timeout=60)
        print(resp)

        reply_content["status"] = "ok"

        await kernel_manager.shutdown_kernel(now=True)
        return reply_content
