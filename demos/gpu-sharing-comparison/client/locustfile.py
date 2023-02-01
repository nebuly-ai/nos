import logging
import os
import random

import torch
import torchvision
import torchvision.transforms as transforms
from locust import HttpUser, task


class Cifar10User(HttpUser):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.log = logging.getLogger()

        # Init test images
        transform = transforms.Compose(
            [transforms.ToTensor(), transforms.Normalize((0.5, 0.5, 0.5), (0.5, 0.5, 0.5))]
        )
        testset = torchvision.datasets.CIFAR10(
            root="./data",
            train=False,
            download=True,
            transform=transform,
        )
        testloader = torch.utils.data.DataLoader(
            testset,
            batch_size=10,
            shuffle=False,
            num_workers=2,
        )
        for data in testloader:
            self.images, _ = data
            break
        self.log.info(f"loaded {len(self.images)} test images")

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
        image_index = random.randint(0, len(self.images) - 1)
        image = self.images[image_index:image_index + 1].tolist()
        payload = (
                '{"inputs":[{"name":"input__0","datatype":"FP32","shape":[1, 3, 32, 32],"data":'
                + f"{image}"
                + "}]}"
        )
        for host in self.target_hosts:
            self.log.info(f"sending request to {host}")
            self.client.post(
                host,
                data=payload,
                headers={"Content-Type": "application/json"},
            )


if __name__ == "__main__":
    u = Cifar10User()
    u.send_request()
