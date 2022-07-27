#!/usr/bin/env python

import json
import os
from importlib.util import module_from_spec, spec_from_file_location

from setuptools import find_packages, setup
from setuptools.command.install import install

_PATH_ROOT = os.path.dirname(__file__)
PACKAGE_NAME = "quick_start"
LIGHTNING_COMPONENT_PREFIX = "demo"


def _load_py_module(fname, pkg=PACKAGE_NAME):
    spec = spec_from_file_location(
        os.path.join(pkg, fname), os.path.join(_PATH_ROOT, pkg, fname)
    )
    py = module_from_spec(spec)
    spec.loader.exec_module(py)
    return py


about = _load_py_module("__about__.py")
setup_tools = _load_py_module("setup_tools.py")
long_description = setup_tools._load_readme_description(
    _PATH_ROOT, homepage=about.__homepage__, ver=about.__version__
)

LIGHTNING_COMPONENT_INFO = {
    "package": PACKAGE_NAME,
    "version": about.__version__,
    "entry_point": LIGHTNING_COMPONENT_PREFIX,
}
# https://packaging.python.org/discussions/install-requires-vs-requirements /
# keep the meta-data here for simplicity in reading this file... it's not obvious
# what happens and to non-engineers they won't know to look in init ...
# the goal of the project is simplicity for researchers, don't want to add too much
# engineer specific practices


class PostInstallCommand(install):
    def run(self):
        install.run(self)
        os.system(
            f"echo Installed lightning component package: {json.dumps(json.dumps(LIGHTNING_COMPONENT_INFO))}"
        )
        print(
            f"echo Installed lightning component package: {json.dumps(json.dumps(LIGHTNING_COMPONENT_INFO))}"
        )


setup(
    name=PACKAGE_NAME,
    version=about.__version__,
    description=about.__docs__,
    author=about.__author__,
    author_email=about.__author_email__,
    url=about.__homepage__,
    download_url="https://github.com/PyTorchLightning/lightning-quick-start",
    license=about.__license__,
    packages=find_packages(exclude=["tests", "docs"]),
    long_description=long_description,
    long_description_content_type="text/markdown",
    include_package_data=True,
    zip_safe=False,
    keywords=["deep learning", "pytorch", "AI"],
    python_requires=">=3.6",
    entry_points={
        "lightning.external_components": [
            f"{LIGHTNING_COMPONENT_PREFIX} = {PACKAGE_NAME}:exported_lightning_components",
        ],
    },
    cmdclass={
        "install": PostInstallCommand,
    },
    setup_requires=["wheel"],
    install_requires=setup_tools._load_requirements(_PATH_ROOT),
)
