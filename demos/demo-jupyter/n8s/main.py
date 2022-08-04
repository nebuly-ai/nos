from n8s.kernel import NebulnetesKernel

if __name__ == '__main__':
    from ipykernel.kernelapp import IPKernelApp

    IPKernelApp.launch_instance(kernel_class=NebulnetesKernel)
