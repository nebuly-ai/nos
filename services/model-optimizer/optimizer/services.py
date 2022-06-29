import abc
import dataclasses
import json
import os
import pathlib
import time
from typing import List

import requests
from azure.identity import ClientSecretCredential
from azure.storage.blob import BlobClient
from loguru import logger

from optimizer import models


class ModelLibrary(abc.ABC):
    def __int__(self, base_uri: str, model_descriptor_uri: str):
        self._base_uri = base_uri
        self._model_descriptor_uri = model_descriptor_uri

    @abc.abstractmethod
    def upload_model(self, model_path: pathlib.Path) -> str:
        pass

    @abc.abstractmethod
    def upload_model_descriptor(self, model_descriptor: models.ModelDescriptor) -> None:
        pass


class AzureModelLibrary(ModelLibrary):
    ENV_VAR_CLIENT_ID = "AZ_CLIENT_ID"
    ENV_VAR_CLIENT_SECRET = "AZ_CLIENT_SECRET"
    ENV_VAR_TENANT_ID = "AZ_TENANT_ID"

    def __init__(self, base_uri: str, model_descriptor_uri: str):
        super().__int__(base_uri, model_descriptor_uri)
        self.__credential = ClientSecretCredential(
            tenant_id=os.environ[self.ENV_VAR_TENANT_ID],
            client_id=os.environ[self.ENV_VAR_CLIENT_ID],
            client_secret=os.environ[self.ENV_VAR_CLIENT_SECRET],
        )

    def _new_blob_client(self, uri: str):
        return BlobClient.from_blob_url(uri, self.__credential, retry_total=0)

    def upload_model(self, model_path: pathlib.Path) -> str:
        with open(model_path, "rb") as data:
            logger.info(f"Uploading model {model_path} to {self._base_uri}")
            self._new_blob_client(self._base_uri).upload_blob(data)
        return self._base_uri  # todo

    def upload_model_descriptor(self, model_descriptor: models.ModelDescriptor):
        logger.info(f"Uploading model descriptor to {self._model_descriptor_uri}")
        model_descriptor_json = json.dumps(dataclasses.asdict(model_descriptor))
        self._new_blob_client(self._model_descriptor_uri).upload_blob(model_descriptor_json)


class ModelLibraryFactory(abc.ABC):
    storage_kind_to_model_library = {
        models.StorageKind.azure: AzureModelLibrary
    }

    @staticmethod
    def new_model_library(storage_kind: models.StorageKind, base_uri: str, model_descriptor_uri: str):
        cls = ModelLibraryFactory.storage_kind_to_model_library.get(storage_kind, None)
        if cls is None:
            raise Exception(f"Storage kind {storage_kind} is not supported")
        return cls(base_uri, model_descriptor_uri)


class ModelOptimizer:
    def __init__(self, working_dir: pathlib.Path):
        """
        Parameters
        ----------
        working_dir: pathlib.Path
            A path pointing to the directory that will be used as working dir by the model optimizer, e.g.
            the directory that will contain all the artifacts and temporary files produced by the optimizer.
        """
        if not working_dir.is_dir():
            raise Exception(f"Invalid working dir: {working_dir} is not a directory")
        self.__working_dir = working_dir

    def _download_model(self, uri: str) -> pathlib.Path:
        logger.info(f"Downloading model from {uri}")
        try:
            resp = requests.get(uri)
            resp.raise_for_status()
            # Todo
            model_path = self.__working_dir / pathlib.Path("model")
            model_path.write_bytes(resp.content)
            logger.debug(f"Model downloaded at {model_path}")
            return model_path
        except requests.HTTPError as e:
            raise Exception(f"Could not download model from {uri} - {e}")

    def optimize_model(self, model_uri: str) -> pathlib.Path:
        """Download the model from the provided uri and optimize it

        Parameters
        ----------
        model_uri: str
            The URI pointing to the model to optimize

        Returns
        -------
        pathlib.Path
            A path pointing to the optimized model
        """
        model_path = self._download_model(model_uri)
        logger.info("Optimizing model with nebullvm 🚀")
        time.sleep(5)
        return model_path


class HardwareSelector:
    def __init__(self, optimization_target: models.OptimizationTarget):
        self.optimization_target = optimization_target

    def select_hardware_for_model(self, model_path: pathlib.Path) -> List[models.SelectedHardware]:
        # Todo
        return []
