FROM python:3.8.13

RUN pip install --upgrade pip
RUN pip install jupyter notebook

RUN useradd -ms /bin/bash nebuly
USER nebuly
WORKDIR /home/nebuly

COPY --chown=nebuly n8s/ code/n8s
COPY pyproject.toml code/
RUN pip install --user code/

RUN mkdir -p kernels/k8s
COPY --chown=nebuly kernel.json kernels/k8s/kernel.json
RUN jupyter kernelspec install kernels/k8s --user