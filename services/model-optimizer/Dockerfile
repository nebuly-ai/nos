FROM python:3.8-slim-buster as builder

COPY pyproject.toml .
COPY poetry.lock .
COPY optimizer optimizer
RUN pip3 install poetry
RUN poetry build


FROM python:3.8-slim-buster

WORKDIR /app

RUN useradd -m -u 1000 worker
USER 1000

COPY --from=builder dist/optimizer-*.whl .
RUN pip3 install --user optimizer-*.whl

ENTRYPOINT ["optimizer"]
