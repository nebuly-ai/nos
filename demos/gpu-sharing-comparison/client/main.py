import requests
from PIL import Image
from prometheus_client import Summary, start_http_server
from transformers import YolosForObjectDetection, YolosImageProcessor

INFERENCE_TIME = Summary("inference_time_seconds", "Time required for running a single inference")


@INFERENCE_TIME.time()
def run_inference(model, inputs):
    model(**inputs)


def benchmark():
    url = 'http://images.cocodataset.org/val2017/000000039769.jpg'
    image = Image.open(requests.get(url, stream=True).raw)

    feature_extractor = YolosImageProcessor.from_pretrained('hustvl/yolos-small')
    model = YolosForObjectDetection.from_pretrained('hustvl/yolos-small')
    model.to("cuda")
    inputs = feature_extractor(images=image, return_tensors="pt").to("cuda")

    while True:
        print("Running inference...")
        run_inference(model, inputs)


if __name__ == "__main__":
    # Start up the server to expose the metrics.
    print("Starting Prometheus server on port 8000...")
    start_http_server(8000)

    # Run benchmark
    print("Running benchmark...")
    benchmark()
