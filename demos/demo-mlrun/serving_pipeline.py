import mlrun

# mlrun: start-code
from kubernetes.client import V1Affinity


def inc(x):
    return x + 1


def mul(x):
    return x * 2


class WithState:
    def __init__(self, name, context, init_val=0):
        self.name = name
        self.context = context
        self.counter = init_val

    def do(self, x):
        self.counter += 1
        print(f"Echo: {self.name}, x: {x}, counter: {self.counter}")
        return x + self.counter


# mlrun: end-code

# The tag mlrun start/end code is used for defining which parts of the script have to be converted to a MLRun function
# when calling the code_to_function function.

if __name__ == "__main__":
    fn = mlrun.code_to_function("simple-graph", kind="job", image="mlrun/mlrun")
    fn.with_node_selection(affinity=V1Affinity())
    graph = fn.set_topology("flow")
    graph.to(name="+1", handler='inc') \
        .to(name="*2", handler='mul') \
        .to(name="(X+counter)", class_name='WithState').respond()
