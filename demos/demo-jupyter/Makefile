.PHONY: build run

build:
	docker build -t nebuly/n8s-jupyter-demo:0.0.1 .

run:
	docker run --rm -it -p 8888:8888 nebuly/n8s-jupyter-demo:0.0.1 jupyter notebook --ip 0.0.0.0
