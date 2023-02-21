FROM python:3.8.16 AS dep-builder

# build wheels for all deps in requirements.txt
COPY requirements.txt /build/requirements.txt
RUN pip wheel -w /build/dist -r /build/requirements.txt

FROM python:3.8.16-slim AS base

WORKDIR /home/nebuly

# copy all wheels from builder and install
COPY --from=dep-builder [ "/build/dist/*.whl", "/install/" ]
RUN pip install --no-index /install/*.whl && rm -rf /install

COPY main.py main.py

RUN chown -R 1000:1000 /home/nebuly
RUN mkdir /.cache ; chown -R 1000:1000 /.cache
USER 1000

EXPOSE "8000"
ENTRYPOINT ["python", "main.py"]


