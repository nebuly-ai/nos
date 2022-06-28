import abc
import os
import pathlib
from typing import List

import requests
from azure.identity import ClientSecretCredential
from azure.storage.blob import BlobClient
from loguru import logger

import models


class ModelLibrary(abc.ABC):
    def __int__(self, base_uri: str, model_descriptor_uri: str):
        self._base_uri = base_uri
        self._model_descriptor_uri = model_descriptor_uri

    @abc.abstractmethod
    def upload_model(self, model_path: pathlib.Path) -> pathlib.Path:
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
        self.__blob_client = self._new_blob_client()

    def _new_blob_client(self):
        credential = ClientSecretCredential(
            tenant_id=os.environ[self.ENV_VAR_TENANT_ID],
            client_id=os.environ[self.ENV_VAR_CLIENT_ID],
            client_secret=os.environ[self.ENV_VAR_CLIENT_SECRET],
        )
        return BlobClient.from_blob_url(self._base_uri, credential)

    def upload_model(self, model_path: pathlib.Path):
        pass

    def upload_model_descriptor(self, model_descriptor: models.ModelDescriptor):
        pass


class ModelLibraryFactory(abc.ABC):
    storage_kind_to_model_library = {
        models.StorageKind.azure: AzureModelLibrary
    }

    @staticmethod
    def new_model_library(storage_kind: models.StorageKind, base_uri: str, model_descriptor_uri: str):
        cls = ModelLibraryFactory.storage_kind_to_model_library[storage_kind]
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

        logger.info(f"Downloading model from {model_uri}")
        resp = requests.get(model_uri)
        resp.raise_for_status()
        # Todo
        model_path = pathlib.Path("model")
        model_path.write_bytes(resp.content)
        logger.info(f"Model downloaded at {model_path}")


class HardwareSelector:
    def __init__(self, optimization_target: models.OptimizationTarget):
        self.optimization_target = optimization_target

    def select_hardware_for_model(self, model_path: pathlib.Path) -> List[models.SelectedHardware]:
        # Todo
        return []


class ModelDescriptorBuilder:
    def __init__(self, model_path: pathlib.Path, selected_hardware: List[models.SelectedHardware]):
        pass

    def build_model_descriptor(self) -> models.ModelDescriptor:
        return models.ModelDescriptor()
