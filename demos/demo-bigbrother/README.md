# Demo Big Brother
The following is a demo of a possible implementation of the tech scenario name 
"Big Brother", where Nebuly provides a platform for continuously monitoring and 
optimizing the steps of the MLOps processes where computing matters.

The platform is supposed to be extensible and to integrate with the common MLOps
platforms and frameworks.

The project is structured as follows:

* [common](common): directory containing the core Nebuly library used for 
analyzing and optimizing workflows, as well as the plugins for integrating with 
the common MLOps frameworks and platforms.
* [demo-kubeflow](demo-kubeflow): demo of an integration with KubeFlow pipelines
* [demo-lightning](demo-lightning): demo of an integration with Lightning Apps

Before running each demo, besides installing the respective requirements.txt, it is
also necessary to install the Nebuly "Bib Brother" library. You can do that as follows:

``pip install -e common/``

