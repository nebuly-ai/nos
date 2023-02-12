import logging
from subprocess import PIPE, Popen, run
import os
import random

import torch
import torchvision
from locust import events
import torchvision.transforms as transforms
from locust import HttpUser, task
import json


class Cifar10User(HttpUser):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.log = logging.getLogger()

        # Init target hosts
        self.target_hosts = []
        for k, v in os.environ.items():
            if k.startswith("TARGET_HOST"):
                self.target_hosts.append(v)
        random.shuffle(self.target_hosts)
        self.log.info(f"target hosts are: {self.target_hosts}")
        if len(self.target_hosts) == 0:
            raise Exception("no hosts provided")

    @task
    def send_request(self):
        image = torch.randn(1, 3, 32, 32).tolist()
        payload = (
            '{"inputs":[{"name":"input__0","datatype":"FP32","shape":[1, 3, 32, 32],"data":'
            + f"{image}"
            + "}]}"
        )
        for host in self.target_hosts:
            self.log.info(f"sending request to {host}")
            resp = self.client.post(
                host,
                data=payload,
                headers={"Content-Type": "application/json"},
            )
            self.log.info(f"response: {resp.text}")
            self.log.info(f"status code: {resp.status_code}")


if __name__ == "__main__":
    u = Cifar10User()
    u.send_request()
